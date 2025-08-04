package sitelib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/bcmk/siren/lib/cmdlib"
)

// IconNames represents all the icons in chic
var IconNames = []string{
	"siren",
	"fanclub",
	"instagram",
	"twitter",
	"onlyfans",
	"amazon",
	"lovense",
	"gift",
	"pornhub",
	"dmca",
	"allmylinks",
	"onemylink",
	"linktree",
	"fancentro",
	"frisk",
	"fansly",
	"throne",
	"mail",
	"snapchat",
	"telegram",
	"whatsapp",
	"youtube",
	"tiktok",
	"reddit",
	"twitch",
	"discord",
	"manyvids",
	"avn",
}

func download(svc *s3.S3, bucketName string, key *string) (*bytes.Buffer, error) {
	out, err := svc.GetObject(&s3.GetObjectInput{
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
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(config.BucketRegion),
		Endpoint:         aws.String(config.BucketEndpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(config.BucketAccessKey, config.BucketSecretKey, ""),
	})
	cmdlib.CheckErr(err)
	svc := s3.New(sess)
	var packs []PackV2
	err = svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.BucketName),
	}, func(page *s3.ListObjectsV2Output, _ bool) bool {
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
		return true
	})
	cmdlib.CheckErr(err)
	sort.Slice(packs, func(i, j int) bool { return packs[i].Timestamp < packs[j].Timestamp })
	return packs
}
