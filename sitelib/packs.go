package sitelib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bcmk/siren/lib/cmdlib"
)

func download(svc *s3.Client, bucketName string, key *string) (*bytes.Buffer, error) {
	ctx := context.Background()

	out, err := svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    key,
	})
	if err != nil {
		return nil, err
	}
	defer func() { cmdlib.CheckErr(out.Body.Close()) }()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, out.Body); err != nil {
		return nil, err
	}
	return &buf, nil
}

// ParsePacksV2 parses icons packs for config V2
func ParsePacksV2(config *Config) []PackV2 {
	ctx := context.Background()
	awscfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(config.BucketRegion),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(config.BucketAccessKey, config.BucketSecretKey, ""),
		),
	)
	cmdlib.CheckErr(err)

	svc := s3.NewFromConfig(awscfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.DisableLogOutputChecksumValidationSkipped = true
		if config.BucketEndpoint != "" {
			o.BaseEndpoint = aws.String(config.BucketEndpoint)
		}
	})

	var packs []PackV2

	p := s3.NewListObjectsV2Paginator(svc, &s3.ListObjectsV2Input{
		Bucket: aws.String(config.BucketName),
	})

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		cmdlib.CheckErr(err)

		for _, obj := range page.Contents {
			if strings.HasSuffix(*obj.Key, "/config_v2.json") {
				fmt.Printf("Parsing %s...\n", *obj.Key)

				buf, err := download(svc, config.BucketName, obj.Key)
				cmdlib.CheckErr(err)

				var pack PackV2
				cmdlib.CheckErr(json.Unmarshal(buf.Bytes(), &pack))

				fullDirPath := filepath.Dir(*obj.Key)
				dirName := filepath.Base(fullDirPath)
				pack.Name = dirName

				packs = append(packs, pack)
			}
		}
	}

	sort.Slice(packs, func(i, j int) bool {
		return packs[i].CreatedAt < packs[j].CreatedAt
	})

	if config.Debug {
		fmt.Println("Parsed packs configuration:")
		out, err := json.MarshalIndent(packs, "", "  ")
		cmdlib.CheckErr(err)
		fmt.Println(string(out))
	}

	return packs
}
