package simplestreams

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/blang/semver/v4"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/ulikunitz/xz"
	"gopkg.in/yaml.v2"
)

var imageTagRegexp = regexp.MustCompile(`^(\S+)/v(.+)$`)

// TagImage represents an image's name and version parsed from a tag
type TagImage struct {
	Name    string
	Version semver.Version
}

func (t TagImage) String() string {
	return t.Name + "/v" + t.Version.String()
}

// ProductID generates a "product ID" for an image
func ProductID(image TagImage, arch string) string {
	return image.Name + ":" + arch
}

// ParseTag attempts to parse a TagImage from a tag string
func ParseTag(tag string) (TagImage, error) {
	t := TagImage{}

	m := imageTagRegexp.FindStringSubmatch(tag)
	if len(m) == 0 {
		return t, ErrInvalidTag
	}

	v, err := semver.Parse(m[2])
	if err != nil {
		return t, fmt.Errorf("failed to parse version: %w", err)
	}

	return TagImage{
		Name:    m[1],
		Version: v,
	}, nil
}

// LoadSHA256 downloads a SHA256 hash from a URL (file in the format used by the
// `sha256sum` command)
func LoadSHA256(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to HTTP GET sha256: %w", err)
	}

	defer resp.Body.Close()
	sum := make([]byte, 64)
	if _, err := io.ReadFull(resp.Body, sum); err != nil {
		return "", fmt.Errorf("failed to read sha256: %w", err)
	}

	return string(sum), nil
}

// LoadImageMeta downloads an lxd.tar.xz file and parses the metadata.yaml file
// within
func LoadImageMeta(url string) (lxdApi.ImageMetadata, string, error) {
	var m lxdApi.ImageMetadata

	resp, err := http.Get(url)
	if err != nil {
		return m, "", fmt.Errorf("failed to HTTP GET archive: %w", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return m, "", fmt.Errorf("failed to read HTTP body: %w", err)
	}

	xzR, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return m, "", fmt.Errorf("failed to create xz reader: %w", err)
	}

	tarR := tar.NewReader(xzR)
	for {
		hdr, err := tarR.Next()
		if err != nil {
			return m, "", fmt.Errorf("failed to read tar entry: %w", err)
		}

		if hdr.Name == "metadata.yaml" {
			break
		}
	}

	if err := yaml.NewDecoder(tarR).Decode(&m); err != nil {
		return m, "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	h := sha256.New()
	h.Write(data)
	sum := fmt.Sprintf("%x", h.Sum(nil))

	return m, sum, nil
}
