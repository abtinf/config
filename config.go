/*
Package `config` is a zero-dependency replacement for using the standard `flag`
package that aims to be simple and eliminate flag parsing boilerplate.

It takes a struct annotated with struct tags and populates it with values inherited
in the following order of precedence:

- Command line arguments
- Environment variables
- Defaults, as specified in the struct tags

The struct tags are as follows:

- `env` - The name of the environment variable to use. This is also used as the
command line flag name.
- `default` - The default value to use if no environment variable or command line
argument is provided.

The following struct field &kinds* are supported: `bool`, `float64`, `int`, `int64`, `string`, `uint`, `uint64`. In addition, the field type `time.Duration` is also supported.

Example usage:

	type C struct {
		HttpHost  string        `env:"HTTP_HOST" default:"localhost"` // Uses default and can be overridden by env or arg
		HttpPort  int           `default:"8080"`                      // Uses default and will not be overridden
		Timeout   time.Duration `default:"30s"`                       // time.Duration types accept values supported by time.ParseDuration
		HttpsPort        // Not populated, as it has no struct tag
	}

	func main() {
		c, err := config.New(os.LookupEnv, os.Args, &C{})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%+v", c)
	}

Notes:
- The default value is optional, but if it is provided then it must be valid. E.g.,
if the field is an integer, the default value must be a valid integer, not an empty string.
*/
package config

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

/*
Populate a struct with its default values, environment variables, and command line arguments.

`lookupenv` is a function to lookup environment variables. If nil, os.LookupEnv is used.

`args` is the command line arguments, typically os.Args. args[0] must be the program name. If nil, os.Args is used.

`c` is pointer to the struct to populate.
*/
func New[T any](lookupenv func(string) (string, bool), args []string, c *T) (*T, error) {
	if lookupenv == nil {
		lookupenv = os.LookupEnv
	}
	if args == nil {
		args = os.Args
	}
	if kind := reflect.ValueOf(c).Kind(); kind != reflect.Pointer {
		return nil, fmt.Errorf("config.New: expected a pointer to a struct, got %s", kind)
	}
	cValue := reflect.ValueOf(c).Elem()
	if kind := cValue.Kind(); kind != reflect.Struct {
		return nil, fmt.Errorf("config.New: expected struct pointer, got %s pointer", kind)
	}

	programName := args[0]
	args = args[1:]
	flagset := buildFlagSet(programName, c)
	if err := flagset.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse command line arguments: %w", err)
	}
	//The `flag` package doesn't expose its internal formal flag set,
	//so visiting every flag is the only way to check which ones were set.
	formalFlagSet := make(map[string]*flag.Flag)
	flagset.Visit(func(f *flag.Flag) {
		formalFlagSet[f.Name] = f
	})

	for i := 0; i < cValue.NumField(); i++ {
		field := cValue.Field(i)
		tag := cValue.Type().Field(i).Tag

		valueFound := false
		valueSource := ""
		valueToSet := ""

		if value, ok := tag.Lookup("default"); ok {
			valueFound = true
			valueToSet = value
			valueSource = "default"
		}
		if varName, ok := tag.Lookup("env"); ok {
			if value, ok := lookupenv(varName); ok {
				valueFound = true
				valueToSet = value
				valueSource = "env"
			}
			if value, ok := formalFlagSet[varName]; ok {
				valueFound = true
				valueToSet = value.Value.String()
				valueSource = "arglist"
			}
		}

		if valueFound {
			if err := setFieldValue(field, valueToSet); err != nil {
				return nil, fmt.Errorf("failed to set field %s to '%s' from %s: %w", field.Type().Name(), valueToSet, valueSource, err)
			}
		}
	}

	return c, nil
}

func buildFlagSet[T any](name string, c *T) *flag.FlagSet {
	flagset := flag.NewFlagSet(name, flag.ContinueOnError)
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := v.Type().Field(i).Tag
		if env := tag.Get("env"); env != "" {
			def := tag.Get("default")
			switch field.Kind() {
			case reflect.Bool:
				v, err := strconv.ParseBool(def)
				if err != nil {
					panic(err)
				}
				flagset.Bool(env, v, "")
			case reflect.Float64:
				v, err := strconv.ParseFloat(def, 64)
				if err != nil {
					panic(err)
				}
				flagset.Float64(env, v, "")
			case reflect.Int:
				v, err := strconv.Atoi(def)
				if err != nil {
					panic(err)
				}
				flagset.Int(env, v, "")
			case reflect.Int64:
				switch field.Interface().(type) {
				case time.Duration:
					v, err := time.ParseDuration(def)
					if err != nil {
						panic(err)
					}
					flagset.Duration(env, v, "")
				default:
					v, err := strconv.ParseInt(def, 10, 64)
					if err != nil {
						panic(err)
					}
					flagset.Int64(env, v, "")
				}
			case reflect.String:
				flagset.String(env, def, "")
			case reflect.Uint:
				v, err := strconv.ParseUint(def, 10, 0)
				if err != nil {
					panic(err)
				}
				flagset.Uint(env, uint(v), "")
			case reflect.Uint64:
				v, err := strconv.ParseUint(def, 10, 64)
				if err != nil {
					panic(err)
				}
				flagset.Uint64(env, v, "")
			}
		}
	}
	return flagset
}

func setFieldValue(field reflect.Value, val string) error {
	switch field.Kind() {
	case reflect.Bool:
		v, err := strconv.ParseBool(val)
		if err != nil {
			panic(err)
		}
		field.SetBool(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			panic(err)
		}
		field.SetFloat(v)
	case reflect.Int:
		v, err := strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
		field.SetInt(int64(v))
	case reflect.Int64:
		switch field.Interface().(type) {
		case time.Duration:
			v, err := time.ParseDuration(val)
			if err != nil {
				panic(err)
			}
			field.SetInt(int64(v))
		default:
			v, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				panic(err)
			}
			field.SetInt(v)
		}
	case reflect.String:
		field.SetString(val)
	case reflect.Uint:
		v, err := strconv.ParseUint(val, 10, 0)
		if err != nil {
			panic(err)
		}
		field.SetUint(v)
	case reflect.Uint64:
		v, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			panic(err)
		}
		field.SetUint(v)
	default:
		return fmt.Errorf("unsupported type %s", field.Kind())
	}
	return nil
}
