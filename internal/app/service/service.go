package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

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
)

type listFn func(*stock.StockParameters) ([]*model.StockDoc, error)

type modelListParams struct {
	ctx         context.Context
	stockParams *stock.StockParameters
	limit       int64
	fn          listFn
}

var stockProp = map[string]int{
	"label":        1,
	"species":      1,
	"plasmid":      1,
	"parent":       1,
	"name":         1,
	"image_map":    1,
	"sequence":     1,
	"plasmid_name": 1,
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
func NewStockService(repo repository.StockRepository, pub message.Publisher, opt ...aphgrpc.Option) *StockService {
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
func (s *StockService) RemoveStock(ctx context.Context, r *stock.StockId) (*empty.Empty, error) {
	e := &empty.Empty{}
	if err := r.Validate(); err != nil {
		return e, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if err := s.repo.RemoveStock(r.Id); err != nil {
		return e, aphgrpc.HandleDeleteError(ctx, err)
	}
	return e, nil
}

func (s *StockService) OboJSONFileUpload(stream stock.StockService_OboJSONFileUploadServer) error {
	in, out := io.Pipe()
	grp := new(errgroup.Group)
	defer in.Close()
	oh := &oboStreamHandler{writer: out, stream: stream}
	grp.Go(oh.Write)
	m, err := s.repo.LoadOboJson(in)
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

func uploadResponse(info *storage.UploadInformation) upload.FileUploadResponse_Status {
	if info.IsCreated {
		return upload.FileUploadResponse_CREATED
	}
	return upload.FileUploadResponse_UPDATED
}

func genNextCursorVal(c string) int64 {
	t, _ := time.Parse("2006-01-02T15:04:05Z", c)
	return t.UnixNano() / 1000000
}

func stockAQLStatement(filter string) (string, error) {
	var vert bool
	var astmt string
	p, err := query.ParseFilterString(filter)
	if err != nil {
		return astmt,
			fmt.Errorf("error in parsing filter string %s", err)
	}
	// need to check if filter contains an item found in strain properties
	for _, n := range p {
		if _, ok := stockProp[n.Field]; ok {
			vert = true
			break
		}
	}
	if vert {
		astmt, err := query.GenAQLFilterStatement(&query.StatementParameters{
			Fmap:    arangodb.FMap,
			Filters: p,
			Vert:    "v",
		})
		if err != nil {
			return astmt,
				fmt.Errorf("error in generating AQL statement %s", err)
		}
	} else {
		astmt, err := query.GenAQLFilterStatement(&query.StatementParameters{
			Fmap:    arangodb.FMap,
			Filters: p,
			Doc:     "s",
		})
		if err != nil {
			return astmt,
				fmt.Errorf("error in generating AQL statement %s", err)
		}
	}
	// if the parsed statement is empty FILTER, just return empty string
	if astmt == "FILTER " {
		astmt = ""
	}
	return astmt, nil
}

func stockModelList(args *modelListParams) ([]*model.StockDoc, error) {
	astmt, err := stockAQLStatement(args.stockParams.Filter)
	if err != nil {
		return []*model.StockDoc{}, aphgrpc.HandleInvalidParamError(args.ctx, err)
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
	if limit > 0 {
		return limit
	}
	return int64(10)
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
		oh.writer.Write(req.Content)
	}
	return nil
}
