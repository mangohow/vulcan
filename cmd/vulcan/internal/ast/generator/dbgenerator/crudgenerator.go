package dbgenerator

import (
	"bytes"
	"github.com/mangohow/mangokit/tools/collection"
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"io"
	"strconv"
	"strings"
)

type CRUDGenFunc func(spec *types.TypeSpec, indexes []int, useNull bool) (string, error)

var ( // TODO
	crudGenFuncMapping = map[string]CRUDGenFunc{
		"Add": func(spec *types.TypeSpec, indexes []int, useNull bool) (string, error) {
			return "", nil
		},
	}
)

func generateCRUDFunc(name string, indexes []int, useNull bool, spec *types.TypeSpec) (string, error) {
	fn, ok := crudGenFuncMapping[name]
	if !ok {
		return "", errors.Errorf("gen func %s is invalid", name)
	}

	return fn(spec, indexes, useNull)
}

// 根据model生成中间代码
func GenerateCRUDFuncsByModel(modelSpecs []*types.TypeSpec) ([]io.Reader, error) {
	readers := make([]io.Reader, 0, len(modelSpecs))
	for _, modelSpec := range modelSpecs {
		items := stream.Filter(modelSpec.Fields, func(param *types.Param) bool {
			return param.Type.Name == "TableProperty" && param.Type.Package.PackagePath == corePackagePath
		})
		if len(items) == 0 {
			continue
		}

		tableProperties := items[0].Type.Tag
		tableName := tableProperties.Get("tableName")
		tableName = strings.TrimSpace(tableName)
		if tableName == "" {
			return nil, errors.Errorf("table name is empty")
		}
		funcNames := tableProperties.Get("gen")
		crudFnNameList := stream.Map(strings.Split(funcNames, ","), func(t string) string {
			return strings.TrimSpace(t)
		})
		if len(crudFnNameList) == 0 {
			continue
		}

		buffer := bytes.NewBuffer(nil)
		for _, curdFnName := range crudFnNameList {
			if curdFnName == "" {
				continue
			}
			name, argStr, found := strings.Cut(curdFnName, "(")
			if found {
				argStr = strings.TrimRight(argStr, ")")
			}
			args := stream.Map(strings.Split(argStr, "|"), func(t string) string {
				return strings.TrimSpace(t)
			})
			var (
				indexes []int
				useNull bool
				err     error
			)
			if len(args) > 0 {
				indexes, err = parseIndexes(args[0])
				if err != nil {
					return nil, err
				}
			}

			if len(args) > 1 {
				ok, err := strconv.ParseBool(args[1])
				if err != nil {
					return nil, err
				}
				useNull = ok
			}

			source, err := generateCRUDFunc(name, indexes, useNull, modelSpec)
			if err != nil {
				return nil, err
			}

			buffer.WriteString(source)
			buffer.WriteString("\n")
		}

		readers = append(readers, buffer)
	}

	return readers, nil
}

func parseIndexes(indexStr string) ([]int, error) {
	set := collection.NewSet[int]()
	indexStrList := strings.Split(indexStr, ",")
	for _, idxStr := range indexStrList {
		before, after, found := strings.Cut(idxStr, "-")
		if !found {
			i, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, err
			}
			set.Add(i)
			continue
		}

		start, err := strconv.Atoi(before)
		if err != nil {
			return nil, err
		}
		end, err := strconv.Atoi(after)
		if err != nil {
			return nil, err
		}
		for ; start <= end; start++ {
			set.Add(start)
		}
	}

	return set.Values(), nil
}
