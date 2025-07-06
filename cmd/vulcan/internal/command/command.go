package command

import (
	"fmt"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"

	"github.com/spf13/cobra"
	"reflect"
	"strconv"
)

type CommandOptions struct {
	File             string `flag:"file" short:"f" usage:"Specify the file to generate the code"`
	Dir              string `flag:"dir" short:"d" usage:"Specify the directory to generate the code"`
	Mode             string `flag:"mode" short:"m" default:"mapper" usage:"Specify the generation mode: [mapper, copy]"`
	DDLGen           bool   `flag:"ddl-gen" usage:"Generate CRUD code based on database ddl statements"`
	StructGen        bool   `flag:"struct-gen" usage:"Generate CRUD code based on the struct model"`
	OutPutPath       string `flag:"output" short:"o" usage:"Specify the path of the generated code"`
	ModelOutputPath  string `flag:"model-output" usage:"Specify the path where the generated structure model code is located"`
	IntermediateCode bool   `flag:"intermediate-code" short:"i" usage:"Whether to generate intermediate code"`
	TablePrefix      string `flag:"table-prefix" default:"t" usage:"Specify table name prefix"`
	ModelSuffix      string `flag:"model-suffix" usage:"Specifies the suffix of the generated model struct"`
	RepoSuffix       string `flag:"repo-suffix" default:"Repo" usage:"Specifies the suffix of the generated database access object"`
	UseNullable      bool   `flag:"use-nullable" default:"true" usage:"When the field can be null, whether to use sql.NullValue as the structure field"`
	Tags             string `flag:"tags" default:"json" usage:"Add tags to the generated model struct and use a comma to separate it"`
}

func BindCommand(cmd *cobra.Command, obj any) (err error) {
	rt := reflect.TypeOf(obj)
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		flag := field.Tag.Get("flag")
		shorthand := field.Tag.Get("short")
		defaultValue := field.Tag.Get("default")
		usage := field.Tag.Get("usage")

		switch field.Type.Kind() {
		case reflect.String:
			cmd.Flags().StringP(flag, shorthand, defaultValue, usage)
		case reflect.Bool:
			defVal := false
			if defaultValue != "" {
				defVal, err = strconv.ParseBool(defaultValue)
				if err != nil {
					return err
				}
			}

			cmd.Flags().BoolP(flag, shorthand, defVal, usage)
		case reflect.Int:
			defVal := 0
			if defaultValue != "" {
				defVal, err = strconv.Atoi(defaultValue)
				if err != nil {
					return err
				}
			}
			cmd.Flags().IntP(flag, shorthand, defVal, usage)
		case reflect.Uint:
			defVal := uint64(0)
			if defaultValue != "" {
				defVal, err = strconv.ParseUint(defaultValue, 10, 64)
				if err != nil {
					return err
				}
			}

			cmd.Flags().UintP(flag, shorthand, uint(defVal), usage)
		default:
			panic(fmt.Sprintf("unsupported command flag type: %s", field.Type.Kind()))
		}
	}

	return nil
}

func BindOptions(cmd *cobra.Command, options any) error {
	rv := reflect.ValueOf(options)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	rt := rv.Type()
	var (
		value any
		err   error
	)
	for i := 0; i < rt.NumField(); i++ {
		fieldType := rt.Field(i)
		flag := fieldType.Tag.Get("flag")
		switch fieldType.Type.Kind() {
		case reflect.String:
			value, err = cmd.Flags().GetString(flag)
		case reflect.Bool:
			value, err = cmd.Flags().GetBool(flag)
		case reflect.Int:
			value, err = cmd.Flags().GetInt(flag)
		case reflect.Uint:
			value, err = cmd.Flags().GetUint(flag)
		default:
			return errors.Errorf("unsupported type: %s", rt.Kind())
		}
		if err != nil {
			return errors.Wrapf(err, "get flag %s error")
		}
		fieldValue := rv.Field(i)
		if fieldValue.CanSet() {
			fieldValue.Set(reflect.ValueOf(value))
		}
	}

	return nil
}
