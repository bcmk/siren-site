// Package sitelib provides a library for siren sites
package sitelib

import (
	"errors"
	"io/fs"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/bcmk/siren/lib/cmdlib"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var checkErr = cmdlib.CheckErr

// IconV2 represents icon from config v2
type IconV2 struct {
	Version int     `json:"version,omitempty"`
	Width   float64 `json:"width,omitempty"`
	Height  float64 `json:"height,omitempty"`
}

// PackV2 represents an icon pack from config v2
type PackV2 struct {
	HumanName            string            `json:"human_name"`
	Scale                int               `json:"scale"`
	ChaturbateIconsScale *int              `json:"chaturbate_icons_scale,omitempty"`
	VGap                 *int              `json:"vgap,omitempty"`
	HGap                 *int              `json:"hgap,omitempty"`
	Disable              bool              `json:"disable"`
	FinalType            string            `json:"final_type"`
	Timestamp            int64             `json:"timestamp"`
	InputType            string            `json:"input_type"`
	Icons                map[string]IconV2 `json:"icons"`

	Name string `json:"-"`
}

// Config represents site or converter config
type Config struct {
	ConnectionString string `mapstructure:"connection_string"`
	ListenAddress    string `mapstructure:"listen_address"`
	BaseURL          string `mapstructure:"base_url"`
	BaseDomain       string `mapstructure:"base_domain"`
	BucketName       string `mapstructure:"bucket_name"`
	BucketRegion     string `mapstructure:"bucket_region"`
	BucketEndpoint   string `mapstructure:"bucket_endpoint"`
	BucketAccessKey  string `mapstructure:"bucket_access_key"`
	BucketSecretKey  string `mapstructure:"bucket_secret_key"`
	BaseBucketURL    string `mapstructure:"base_bucket_url"`
	AssetsBucketURL  string `mapstructure:"assets_bucket_url"`
	Debug            bool   `mapstructure:"debug"`
}

type configFile struct {
	name     string
	required bool
}

func bindEnvForStructType(v *viper.Viper, t reflect.Type, prefix string, bindPrimitiveMaps bool) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := f.Tag.Get("mapstructure")
			if tag == "" || tag == "-" {
				continue
			}
			key := tag
			if prefix != "" {
				key = prefix + "." + tag
			}
			bindEnvForStructType(v, f.Type, key, bindPrimitiveMaps)
		}
	case reflect.Map:
		if !bindPrimitiveMaps {
			return
		}
		k, e := t.Key(), t.Elem()
		for e.Kind() == reflect.Ptr {
			e = e.Elem()
		}
		if k.Kind() == reflect.String && isPrimitiveKind(e.Kind()) {
			_ = v.BindEnv(prefix)
		}
	default:
		_ = v.BindEnv(prefix)
	}
}

func isPrimitiveKind(k reflect.Kind) bool {
	switch k {
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.String:

		return true
	default:
		return false
	}
}

func stringToSliceHookFunc(sep string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.SliceOf(f) {
			return data, nil
		}

		raw := data.(string)
		if raw == "" {
			return []string{}, nil
		}

		result := strings.Split(raw, sep)
		for k, v := range result {
			result[k] = strings.TrimLeft(v, " ")
		}
		return result, nil
	}
}

var cfgPath = pflag.StringP("config", "c", "", "path to a config file (overrides default search)")

// ReadConfig reads config file and parses it
func ReadConfig() *Config {
	pflag.Parse()

	var configFiles []configFile
	if *cfgPath != "" {
		configFiles = []configFile{{*cfgPath, true}}
	} else {
		configFiles = []configFile{
			{"config.yaml", true},
			{"config.dev.ignore.yaml", false},
		}
	}

	v := viper.New()
	v.SetConfigType("yaml")

	for _, f := range configFiles {
		v.SetConfigFile(f.name)
		if err := v.MergeInConfig(); err != nil {
			if errors.Is(err, fs.ErrNotExist) && !f.required {
				log.Printf("skip config %q", f.name)
				continue
			}
			log.Fatalf("error reading %q: %v", f.name, err)
		}
		log.Printf("successfully read config %q", f.name)
	}

	v.SetEnvPrefix("XRN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	cfg := &Config{}
	bindEnvForStructType(v, reflect.TypeOf(cfg), "", false)
	checkErr(v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnused = true
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			stringToSliceHookFunc(","),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			mapstructure.TextUnmarshallerHookFunc(),
		)
	}))

	return cfg
}
