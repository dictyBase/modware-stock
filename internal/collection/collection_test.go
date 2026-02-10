package collection

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// Test types for Map function tests
type testPerson struct {
	Name string
	Age  int
}

type testPersonDTO struct {
	FullName string
	Years    int
}

func TestMap_EmptyAndSingle(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		input := []int{}
		result := Map(input, strconv.Itoa)

		require.Empty(t, result)
		require.Len(t, result, 0)
		require.NotNil(t, result)
	})

	t.Run("single element", func(t *testing.T) {
		t.Parallel()

		input := []int{42}
		result := Map(input, strconv.Itoa)

		require.Len(t, result, 1)
		require.Equal(t, "42", result[0])
	})

	t.Run("multiple elements", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		result := Map(input, strconv.Itoa)

		require.Len(t, result, 5)
		require.Equal(t, []string{"1", "2", "3", "4", "5"}, result)
	})
}

func TestMap_BasicTransformations(t *testing.T) {
	t.Parallel()

	t.Run("int to string transformation", func(t *testing.T) {
		t.Parallel()

		input := []int{10, 20, 30, 40, 50}
		result := Map(input, func(num int) string {
			return fmt.Sprintf("number_%d", num)
		})

		require.Len(t, result, 5)
		require.Equal(t, "number_10", result[0])
		require.Equal(t, "number_20", result[1])
		require.Equal(t, "number_30", result[2])
		require.Equal(t, "number_40", result[3])
		require.Equal(t, "number_50", result[4])
	})

	t.Run("string to int transformation", func(t *testing.T) {
		t.Parallel()

		input := []string{"1", "2", "3", "4", "5"}
		result := Map(input, func(str string) int {
			num, _ := strconv.Atoi(str)
			return num
		})

		require.Len(t, result, 5)
		require.Equal(t, []int{1, 2, 3, 4, 5}, result)
	})

	t.Run("string to length transformation", func(t *testing.T) {
		t.Parallel()

		input := []string{"a", "ab", "abc", "abcd"}
		result := Map(input, func(str string) int {
			return len(str)
		})

		require.Len(t, result, 4)
		require.Equal(t, []int{1, 2, 3, 4}, result)
	})

	t.Run("string uppercase transformation", func(t *testing.T) {
		t.Parallel()

		input := []string{"hello", "world", "go", "lang"}
		result := Map(input, strings.ToUpper)

		require.Len(t, result, 4)
		require.Equal(t, []string{"HELLO", "WORLD", "GO", "LANG"}, result)
	})
}

func TestMap_StructTransformations(t *testing.T) {
	t.Parallel()

	t.Run("struct to struct transformation", func(t *testing.T) {
		t.Parallel()

		input := []testPerson{
			{Name: "Alice", Age: 30},
			{Name: "Bob", Age: 25},
			{Name: "Charlie", Age: 35},
		}

		result := Map(input, func(person testPerson) testPersonDTO {
			return testPersonDTO{
				FullName: person.Name,
				Years:    person.Age,
			}
		})

		require.Len(t, result, 3)
		require.Equal(t, "Alice", result[0].FullName)
		require.Equal(t, 30, result[0].Years)
		require.Equal(t, "Bob", result[1].FullName)
		require.Equal(t, 25, result[1].Years)
		require.Equal(t, "Charlie", result[2].FullName)
		require.Equal(t, 35, result[2].Years)
	})

	t.Run("struct to string transformation", func(t *testing.T) {
		t.Parallel()

		input := []testPerson{
			{Name: "Alice", Age: 30},
			{Name: "Bob", Age: 25},
		}

		result := Map(input, func(person testPerson) string {
			return fmt.Sprintf("%s is %d years old", person.Name, person.Age)
		})

		require.Len(t, result, 2)
		require.Equal(t, "Alice is 30 years old", result[0])
		require.Equal(t, "Bob is 25 years old", result[1])
	})
}

func TestMap_PointerTransformations(t *testing.T) {
	t.Parallel()

	t.Run("pointer to value transformation", func(t *testing.T) {
		t.Parallel()

		val1, val2, val3 := 1, 2, 3
		input := []*int{&val1, &val2, &val3}

		result := Map(input, func(ptr *int) int {
			return *ptr
		})

		require.Len(t, result, 3)
		require.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("value to pointer transformation", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3}
		result := Map(input, func(num int) *int {
			val := num
			return &val
		})

		require.Len(t, result, 3)
		require.Equal(t, 1, *result[0])
		require.Equal(t, 2, *result[1])
		require.Equal(t, 3, *result[2])
	})
}

func TestMap_ComplexTransformations(t *testing.T) {
	t.Parallel()

	t.Run("transformation with modification", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		result := Map(input, func(num int) int {
			return num * 2
		})

		require.Len(t, result, 5)
		require.Equal(t, []int{2, 4, 6, 8, 10}, result)
	})

	t.Run("bool to string transformation", func(t *testing.T) {
		t.Parallel()

		input := []bool{true, false, true, false}
		result := Map(input, func(b bool) string {
			if b {
				return "yes"
			}
			return "no"
		})

		require.Len(t, result, 4)
		require.Equal(t, []string{"yes", "no", "yes", "no"}, result)
	})

	t.Run("complex number transformation", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3}
		result := Map(input, func(num int) int {
			return num*num + num*2 + 1
		})

		require.Len(t, result, 3)
		require.Equal(t, 4, result[0])  // 1*1 + 1*2 + 1 = 4
		require.Equal(t, 9, result[1])  // 2*2 + 2*2 + 1 = 9
		require.Equal(t, 16, result[2]) // 3*3 + 3*2 + 1 = 16
	})

	t.Run("maintains original slice unchanged", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3}
		original := make([]int, len(input))
		copy(original, input)

		Map(input, strconv.Itoa)

		require.Equal(t, original, input, "Original slice should remain unchanged")
	})
}

func TestFilterFlags_EmptyAndAll(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{}
		result := FilterFlags(input, func(_ cli.Flag) bool {
			return true
		})

		require.Empty(t, result)
		require.Len(t, result, 0)
	})

	t.Run("all flags pass filter", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1"},
			cli.StringFlag{Name: "flag2"},
			cli.StringFlag{Name: "flag3"},
		}

		result := FilterFlags(input, func(_ cli.Flag) bool {
			return true
		})

		require.Len(t, result, 3)
		require.Equal(t, input, result)
	})

	t.Run("no flags pass filter", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1"},
			cli.StringFlag{Name: "flag2"},
			cli.StringFlag{Name: "flag3"},
		}

		result := FilterFlags(input, func(_ cli.Flag) bool {
			return false
		})

		require.Empty(t, result)
		require.Len(t, result, 0)
	})
}

func TestFilterFlags_Partial(t *testing.T) {
	t.Parallel()

	t.Run("some flags pass filter", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "include1"},
			cli.StringFlag{Name: "exclude1"},
			cli.StringFlag{Name: "include2"},
			cli.StringFlag{Name: "exclude2"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			return strings.HasPrefix(flag.GetName(), "include")
		})

		require.Len(t, result, 2)
		require.Equal(t, "include1", result[0].GetName())
		require.Equal(t, "include2", result[1].GetName())
	})

	t.Run("single flag matches", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1"},
			cli.StringFlag{Name: "flag2"},
			cli.StringFlag{Name: "target"},
			cli.StringFlag{Name: "flag3"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			return flag.GetName() == "target"
		})

		require.Len(t, result, 1)
		require.Equal(t, "target", result[0].GetName())
	})
}

func TestFilterFlags_ByType(t *testing.T) {
	t.Parallel()

	t.Run("filter by flag type - StringFlag only", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "string1"},
			cli.BoolFlag{Name: "bool1"},
			cli.StringFlag{Name: "string2"},
			cli.IntFlag{Name: "int1"},
			cli.StringFlag{Name: "string3"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			_, ok := flag.(cli.StringFlag)
			return ok
		})

		require.Len(t, result, 3)
		_, ok := result[0].(cli.StringFlag)
		require.True(t, ok)
		_, ok = result[1].(cli.StringFlag)
		require.True(t, ok)
		_, ok = result[2].(cli.StringFlag)
		require.True(t, ok)
	})

	t.Run("filter by flag type - BoolFlag only", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "string1"},
			cli.BoolFlag{Name: "bool1"},
			cli.BoolFlag{Name: "bool2"},
			cli.IntFlag{Name: "int1"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			_, ok := flag.(cli.BoolFlag)
			return ok
		})

		require.Len(t, result, 2)
		require.Equal(t, "bool1", result[0].GetName())
		require.Equal(t, "bool2", result[1].GetName())
	})

	t.Run("filter by flag type - IntFlag only", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "string1"},
			cli.IntFlag{Name: "int1"},
			cli.BoolFlag{Name: "bool1"},
			cli.IntFlag{Name: "int2"},
			cli.IntFlag{Name: "int3"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			_, ok := flag.(cli.IntFlag)
			return ok
		})

		require.Len(t, result, 3)
		require.Equal(t, "int1", result[0].GetName())
		require.Equal(t, "int2", result[1].GetName())
		require.Equal(t, "int3", result[2].GetName())
	})
}

func TestFilterFlags_ByNamePattern(t *testing.T) {
	t.Parallel()

	t.Run("filter by name pattern", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "db-host"},
			cli.StringFlag{Name: "db-port"},
			cli.StringFlag{Name: "db-name"},
			cli.StringFlag{Name: "api-key"},
			cli.StringFlag{Name: "api-secret"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			return strings.HasPrefix(flag.GetName(), "db-")
		})

		require.Len(t, result, 3)
		require.Equal(t, "db-host", result[0].GetName())
		require.Equal(t, "db-port", result[1].GetName())
		require.Equal(t, "db-name", result[2].GetName())
	})

	t.Run("filter by name suffix", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "username"},
			cli.StringFlag{Name: "password"},
			cli.BoolFlag{Name: "verbose"},
			cli.IntFlag{Name: "timeout"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			return strings.HasSuffix(flag.GetName(), "name")
		})

		require.Len(t, result, 1)
		require.Equal(t, "username", result[0].GetName())
	})

	t.Run("filter by name contains", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "arangodb-host"},
			cli.StringFlag{Name: "arangodb-port"},
			cli.StringFlag{Name: "nats-host"},
			cli.StringFlag{Name: "nats-port"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			return strings.Contains(flag.GetName(), "arango")
		})

		require.Len(t, result, 2)
		require.Equal(t, "arangodb-host", result[0].GetName())
		require.Equal(t, "arangodb-port", result[1].GetName())
	})
}

func TestFilterFlags_ComplexPredicates(t *testing.T) {
	t.Parallel()

	t.Run("mixed flag types with complex predicate", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "config"},
			cli.BoolFlag{Name: "verbose"},
			cli.IntFlag{Name: "port"},
			cli.StringFlag{Name: "host"},
			cli.BoolFlag{Name: "debug"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			_, isBool := flag.(cli.BoolFlag)
			return isBool || flag.GetName() == "config"
		})

		require.Len(t, result, 3)
		require.Equal(t, "config", result[0].GetName())
		require.Equal(t, "verbose", result[1].GetName())
		require.Equal(t, "debug", result[2].GetName())
	})

	t.Run("complex flag with EnvVar", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1", EnvVar: "ENV1"},
			cli.StringFlag{Name: "flag2", EnvVar: "ENV2"},
			cli.StringFlag{Name: "flag3"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			if strFlag, ok := flag.(cli.StringFlag); ok {
				return strFlag.EnvVar != ""
			}
			return false
		})

		require.Len(t, result, 2)
		require.Equal(t, "flag1", result[0].GetName())
		require.Equal(t, "flag2", result[1].GetName())
	})

	t.Run("filter flags with usage text", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1", Usage: "Important flag"},
			cli.StringFlag{Name: "flag2", Usage: ""},
			cli.StringFlag{Name: "flag3", Usage: "Important setting"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			if strFlag, ok := flag.(cli.StringFlag); ok {
				return strings.Contains(strFlag.Usage, "Important")
			}
			return false
		})

		require.Len(t, result, 2)
		require.Equal(t, "flag1", result[0].GetName())
		require.Equal(t, "flag3", result[1].GetName())
	})
}

func TestFilterFlags_Behavior(t *testing.T) {
	t.Parallel()

	t.Run("maintains order of filtered flags", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "z-flag"},
			cli.StringFlag{Name: "a-flag"},
			cli.StringFlag{Name: "m-flag"},
			cli.StringFlag{Name: "b-flag"},
		}

		result := FilterFlags(input, func(flag cli.Flag) bool {
			name := flag.GetName()
			return name == "z-flag" || name == "m-flag" || name == "b-flag"
		})

		require.Len(t, result, 3)
		require.Equal(t, "z-flag", result[0].GetName())
		require.Equal(t, "m-flag", result[1].GetName())
		require.Equal(t, "b-flag", result[2].GetName())
	})

	t.Run("nil predicate behavior is not applicable - requires function", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1"},
		}

		result := FilterFlags(input, func(_ cli.Flag) bool {
			return false
		})

		require.Empty(t, result)
	})

	t.Run("original slice unchanged after filtering", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "flag1"},
			cli.StringFlag{Name: "flag2"},
			cli.StringFlag{Name: "flag3"},
		}

		originalLen := len(input)
		originalNames := make([]string, len(input))
		for idx, flag := range input {
			originalNames[idx] = flag.GetName()
		}

		FilterFlags(input, func(flag cli.Flag) bool {
			return flag.GetName() == "flag1"
		})

		require.Len(t, input, originalLen)
		for idx, flag := range input {
			require.Equal(t, originalNames[idx], flag.GetName())
		}
	})
}

func TestFilterFlags_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("very large slice", func(t *testing.T) {
		t.Parallel()

		// Create a large slice of flags
		input := make([]cli.Flag, 1000)
		for idx := 0; idx < 1000; idx++ {
			input[idx] = cli.StringFlag{Name: fmt.Sprintf("flag%d", idx)}
		}

		// Filter to only even-numbered flags
		result := FilterFlags(input, func(flag cli.Flag) bool {
			name := flag.GetName()
			// Extract number from "flagN"
			var num int
			_, _ = fmt.Sscanf(name, "flag%d", &num)
			return num%2 == 0
		})

		require.Len(t, result, 500)
		require.Equal(t, "flag0", result[0].GetName())
		require.Equal(t, "flag998", result[499].GetName())
	})

	t.Run("single element slice - match", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "single"},
		}

		result := FilterFlags(input, func(_ cli.Flag) bool {
			return true
		})

		require.Len(t, result, 1)
		require.Equal(t, "single", result[0].GetName())
	})

	t.Run("single element slice - no match", func(t *testing.T) {
		t.Parallel()

		input := []cli.Flag{
			cli.StringFlag{Name: "single"},
		}

		result := FilterFlags(input, func(_ cli.Flag) bool {
			return false
		})

		require.Empty(t, result)
	})
}
