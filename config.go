/*
Package config implements an configuration object. that inherits as follows:
- Defaults
- Environment variables
- Command line arguments

The names of the environment variables and command line flags, as well as the default values, are set using struct tags.
*/
package config

import (
	"flag"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
)

type Config struct {
	HttpHost string `env:"HTTP_HOST" default:"localhost"`
	HttpPort int    `env:"HTTP_PORT" default:"8080"`

	ExampleBasicAuthUser     string `env:"USER" default:"gonfoot" redact:"true"`
	ExampleBasicAuthPassword string `env:"PASSWORD" default:"footgon" redact:"true"`
}

func (c *Config) LogValue() slog.Value {
	var values []slog.Attr
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		val := fmt.Sprintf("%v", v.Field(i))
		tag := v.Type().Field(i).Tag
		if tag.Get("redact") == "true" {
			val = "REDACTED"
		}
		values = append(values, slog.String(
			tag.Get("env"),
			val,
		))
	}
	return slog.GroupValue(values...)
}

func buildFlagSet(name string, c *Config) *flag.FlagSet {
	flagset := flag.NewFlagSet(name, flag.ContinueOnError)
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := v.Type().Field(i).Tag
		if env := tag.Get("env"); env != "" {
			def := tag.Get("default")
			switch field.Kind() {
			case reflect.String:
				flagset.String(env, def, "")
			case reflect.Int:
				v, err := strconv.Atoi(def)
				if err != nil {
					panic(err)
				}
				flagset.Int(env, v, "")
			case reflect.Bool:
				v, err := strconv.ParseBool(def)
				if err != nil {
					panic(err)
				}
				flagset.Bool(env, v, "")
			}
		}
	}
	return flagset
}

func New(lookupenv func(string) (string, bool), args []string) (*Config, error) {
	c := Config{}

	flagset := buildFlagSet(args[0], &c)
	if len(args) > 1 {
		if err := flagset.Parse(args[1:]); err != nil {
			return nil, err
		}
	}
	formalFlagSet := make(map[string]*flag.Flag)
	flagset.Visit(func(f *flag.Flag) {
		formalFlagSet[f.Name] = f
	})

	v := reflect.ValueOf(&c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := v.Type().Field(i).Tag
		if env := tag.Get("env"); env != "" {
			val := tag.Get("default")
			if lookupenv != nil {
				if v, ok := lookupenv(env); ok {
					val = v
				}
			}
			if f, ok := formalFlagSet[env]; ok {
				val = f.Value.String()
			}
			if err := setFieldValue(field, val); err != nil {
				return nil, err
			}
		}
	}
	return &c, nil
}

func setFieldValue(field reflect.Value, val string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(val)
	case reflect.Int:
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		field.SetInt(int64(v))
	case reflect.Bool:
		v, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		field.SetBool(v)
	default:
		return fmt.Errorf("unsupported type %s", field.Kind())
	}
	return nil
}
