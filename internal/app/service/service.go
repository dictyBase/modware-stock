package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	E "github.com/IBM/fp-go/either"
	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	ORD "github.com/IBM/fp-go/ord"
	S "github.com/IBM/fp-go/string"
	T "github.com/IBM/fp-go/tuple"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/arangomanager/query"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/upload"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/dictyBase/modware-stock/internal/message"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
	"golang.org/x/sync/errgroup"
	empty "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type listFn func(*stock.StockParameters) ([]*model.StockDoc, error)

type modelListParams struct {
	ctx         context.Context
	stockParams *stock.StockParameters
	limit       int64
	fn          listFn
}

// StockService is the container for managing stock service
// definition
type StockService struct {
	*aphgrpc.Service
	repo      repository.StockRepository
	publisher message.Publisher
	stock.UnimplementedStockServiceServer
}

func defaultOptions() *aphgrpc.ServiceOptions {
	return &aphgrpc.ServiceOptions{Resource: "stock"}
}

// NewStockService is the constructor for creating a new instance of StockService
func NewStockService(
	repo repository.StockRepository,
	pub message.Publisher,
	opt ...aphgrpc.Option,
) *StockService {
	s := defaultOptions()
	for _, optfn := range opt {
		optfn(s)
	}
	srv := &aphgrpc.Service{}
	aphgrpc.AssignFieldsToStructs(s, srv)
	return &StockService{
		Service:   srv,
		repo:      repo,
		publisher: pub,
	}
}

// RemoveStock removes an existing stock
func (s *StockService) RemoveStock(
	ctx context.Context,
	r *stock.StockId,
) (*empty.Empty, error) {
	e := &empty.Empty{}
	if err := r.Validate(); err != nil {
		return e, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if err := s.repo.RemoveStock(r.Id); err != nil {
		return e, aphgrpc.HandleDeleteError(ctx, err)
	}
	return e, nil
}

func (s *StockService) OboJSONFileUpload(
	stream stock.StockService_OboJSONFileUploadServer,
) error {
	in, out := io.Pipe()
	grp := new(errgroup.Group)
	defer in.Close()
	oh := &oboStreamHandler{writer: out, stream: stream}
	grp.Go(oh.Write)
	m, err := s.repo.LoadOboJSON(in)
	if err != nil {
		return aphgrpc.HandleGenericError(context.Background(), err)
	}
	if err := grp.Wait(); err != nil {
		return aphgrpc.HandleGenericError(context.Background(), err)
	}
	return stream.SendAndClose(&upload.FileUploadResponse{
		Status: uploadResponse(m),
		Msg:    "obojson file is uploaded",
	})
}

func uploadResponse(
	info *storage.UploadInformation,
) upload.FileUploadResponse_Status {
	if info.IsCreated {
		return upload.FileUploadResponse_CREATED
	}
	return upload.FileUploadResponse_UPDATED
}

func genNextCursorVal(pts *timestamppb.Timestamp) int64 {
	tstmp := aphgrpc.ProtoTimeStamp(pts)
	return tstmp.UnixMilli()
}

// Use fp-go string API
var isEmptyString = S.IsEmpty

func isEmptyFilterStmt(stmt string) bool { return stmt == "FILTER " }

// Use fp-go ord API for numeric comparison
var (
	int64Ord   = ORD.FromStrictCompare[int64]()
	isPositive = ORD.Gt(int64Ord)(int64(0))
)

var (
	parseFilterString     = E.Eitherize1(query.ParseFilterString)
	genQualifiedAQLFilter = E.Eitherize2(query.GenQualifiedAQLFilterStatement)
)

func normalizeFilterStmt(stmt string) string {
	return F.Pipe1(
		stmt,
		F.Ternary(isEmptyFilterStmt, F.Constant1[string](""), F.Identity[string]),
	)
}

func generateAQL(filters []*query.Filter) E.Either[error, string] {
	return F.Pipe1(
		genQualifiedAQLFilter(arangodb.FMap, filters),
		E.MapLeft[string](fperrors.OnError("error in generating AQL statement")),
	)
}

func emptyFilterRight(_ string) E.Either[error, string] {
	return E.Right[error]("")
}

func parseAndGenerateAQL(fstr string) E.Either[error, string] {
	return F.Pipe4(
		fstr,
		parseFilterString,
		E.MapLeft[[]*query.Filter](
			fperrors.OnError("error in parsing filter string"),
		),
		E.Chain(generateAQL),
		E.Map[error](normalizeFilterStmt),
	)
}

func stockAQLStatementEither(fstr string) E.Either[error, string] {
	return F.Pipe1(
		fstr,
		F.Ternary(isEmptyString, emptyFilterRight, parseAndGenerateAQL),
	)
}

func stockAQLStatement(fstr string) (string, error) {
	result := F.Pipe1(
		stockAQLStatementEither(fstr),
		E.Fold(
			func(err error) T.Tuple2[string, error] { return T.MakeTuple2("", err) },
			func(stmt string) T.Tuple2[string, error] {
				return T.MakeTuple2[string, error](stmt, nil)
			},
		),
	)
	return result.F1, result.F2
}

func stockModelList(args *modelListParams) ([]*model.StockDoc, error) {
	astmt, err := stockAQLStatement(args.stockParams.Filter)
	if err != nil {
		return []*model.StockDoc{}, aphgrpc.HandleInvalidParamError(
			args.ctx,
			err,
		)
	}
	mc, err := args.fn(&stock.StockParameters{
		Cursor: args.stockParams.Cursor,
		Limit:  args.limit,
		Filter: astmt,
	})
	if err != nil {
		return mc, aphgrpc.HandleGetError(args.ctx, err)
	}
	if len(mc) == 0 {
		return mc,
			aphgrpc.HandleNotFoundError(
				args.ctx, errors.New("could not find any strains"),
			)
	}
	return mc, nil
}

func limitVal(limit int64) int64 {
	return F.Pipe1(
		limit,
		F.Ternary(
			isPositive,
			F.Identity[int64],
			F.Constant1[int64](int64(10)),
		),
	)
}

type oboStreamHandler struct {
	writer *io.PipeWriter
	stream stock.StockService_OboJSONFileUploadServer
}

func (oh *oboStreamHandler) Write() error {
	defer oh.writer.Close()
	for {
		req, err := oh.stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		_, err = oh.writer.Write(req.Content)
		if err != nil {
			return fmt.Errorf("error in writing context %s", err)
		}
	}
	return nil
}
