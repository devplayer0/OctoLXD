package simplestreams

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/go-github/v33/github"
	lxdApi "github.com/lxc/lxd/shared/api"
	ss "github.com/lxc/lxd/shared/simplestreams"
	log "github.com/sirupsen/logrus"
)

// PathPattern represents a mux.Router pattern for file paths
const PathPattern = "/images/{name}/{arch}/v{version}/{file}"

const (
	nameDateLayout = "20060102"
	eolLayout      = "2006-01-02"
)

var (
	metaRegexp = regexp.MustCompile(`^lxd\.(.+)\.tar\.xz$`)

	rootFSRegexp    = regexp.MustCompile(`^rootfs\.(.+)\.squashfs$`)
	rootFSSumRegexp = regexp.MustCompile(`^rootfs\.(.+)\.squashfs.sha256$`)

	combinedSumRegexp = regexp.MustCompile(`^combined\.(.+)\.sha256$`)
)

// SimpleStreams represents a proxy which speaks simplestreams for a GitHub
// repo's releases
type SimpleStreams struct {
	client *github.Client
}

// NewSimpleStreams creates a new simplestreams proxy
func NewSimpleStreams() *SimpleStreams {
	return &SimpleStreams{
		client: github.NewClient(nil),
	}
}

func (s *SimpleStreams) listAllReleases(ctx context.Context, owner, repo string) ([]*github.RepositoryRelease, error) {
	opt := &github.ListOptions{
		PerPage: 50,
	}

	var all []*github.RepositoryRelease
	for {
		items, resp, err := s.client.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return all, nil
}

// GenerateStream generates a struct of the simplestreams index.json (for a given
// GitHub repo)
func (s *SimpleStreams) GenerateStream(ctx context.Context, owner, repo string) (ss.Stream, error) {
	stream := ss.Stream{
		Index:  map[string]ss.StreamIndex{},
		Format: "index:1.0",
	}

	rels, err := s.listAllReleases(ctx, owner, repo)
	if err != nil {
		return stream, fmt.Errorf("failed to list releases for %v/%v: %w", owner, repo, err)
	}

	var ps []string
	for _, rel := range rels {
		image, err := ParseTag(*rel.TagName)
		if err != nil {
			log.WithField("tag", *rel.TagName).WithError(err).Warn("Failed to parse release")
			continue
		}

		var archs []string
		for _, ass := range rel.Assets {
			if m := metaRegexp.FindStringSubmatch(*ass.Name); len(m) != 0 {
				archs = append(archs, m[1])
			}
		}

		for _, arch := range archs {
			ps = append(ps, ProductID(image, arch))
		}
	}
	stream.Index["images"] = ss.StreamIndex{
		DataType: "image-downloads",
		Path:     "streams/v1/images.json",
		Products: ps,
	}

	return stream, nil
}

type versionArchInfo struct {
	meta     *lxdApi.ImageMetadata
	metaSize int64
	metaSum  string

	rootFSURL  string
	rootFSSize int64
	rootFSSum  string

	combinedSum string
}

func (s *SimpleStreams) releaseToProducts(products map[string]ss.Product, image TagImage, rel *github.RepositoryRelease) error {
	var err error

	infos := map[string]*versionArchInfo{}
	archMatch := func(r *regexp.Regexp, n string) string {
		if m := r.FindStringSubmatch(n); len(m) != 0 {
			if _, ok := infos[m[1]]; !ok {
				infos[m[1]] = &versionArchInfo{}
			}
			return m[1]
		}

		return ""
	}
	for _, ass := range rel.Assets {
		n := *ass.Name

		if arch := archMatch(metaRegexp, n); arch != "" {
			meta, sum, err := LoadImageMeta(*ass.BrowserDownloadURL)
			if err != nil {
				return fmt.Errorf("failed to load image metadata from release (%v): %w", arch, err)
			}

			infos[arch].meta = &meta
			infos[arch].metaSize = int64(*ass.Size)
			infos[arch].metaSum = sum
		} else if arch := archMatch(rootFSRegexp, n); arch != "" {
			infos[arch].rootFSURL = *ass.BrowserDownloadURL
			infos[arch].rootFSSize = int64(*ass.Size)
		} else if arch := archMatch(rootFSSumRegexp, n); arch != "" {
			infos[arch].rootFSSum, err = LoadSHA256(*ass.BrowserDownloadURL)
			if err != nil {
				return fmt.Errorf("failed to load rootfs checksum from release (%v): %w", arch, err)
			}
		} else if arch := archMatch(combinedSumRegexp, n); arch != "" {
			infos[arch].combinedSum, err = LoadSHA256(*ass.BrowserDownloadURL)
			if err != nil {
				return fmt.Errorf("failed to load combined checksum from release (%v): %w", arch, err)
			}
		}
	}

	for arch, info := range infos {
		if info.meta == nil || info.rootFSURL == "" || info.rootFSSum == "" || info.combinedSum == "" {
			return ErrIncompleteRelease
		}

		pid := ProductID(image, arch)
		if _, ok := products[pid]; !ok {
			p := &ss.Product{
				Aliases:      image.Name,
				Architecture: arch,
				Versions:     map[string]ss.ProductVersion{},
			}

			if os, ok := info.meta.Properties["os"]; ok {
				p.OperatingSystem = os
			}
			if rel, ok := info.meta.Properties["release"]; ok {
				p.Release = rel
				p.ReleaseTitle = rel
			}
			if version, ok := info.meta.Properties["version"]; ok {
				p.Version = version
			}
			if variant, ok := info.meta.Properties["variant"]; ok {
				p.Variant = variant
			}

			if info.meta.ExpiryDate != 0 {
				p.SupportedEOL = time.Unix(info.meta.ExpiryDate, 0).Format(eolLayout)
			}

			products[pid] = *p
		}

		basePath := fmt.Sprintf("images/%v/%v/v%v", image.Name, arch, image.Version)
		versionID := fmt.Sprintf("%v_v%v", time.Unix(info.meta.CreationDate, 0).Format(nameDateLayout), image.Version)
		description, _ := info.meta.Properties["description"]
		products[pid].Versions[versionID] = ss.ProductVersion{
			Items: map[string]ss.ProductVersionItem{
				"lxd.tar.xz": {
					FileType:              "lxd.tar.xz",
					HashSha256:            info.metaSum,
					Size:                  info.metaSize,
					Path:                  basePath + "/lxd.tar.xz",
					LXDHashSha256SquashFs: info.combinedSum,
				},
				"rootfs.squashfs": {
					FileType:   "squashfs",
					HashSha256: info.rootFSSum,
					Size:       info.rootFSSize,
					Path:       basePath + "/rootfs.squashfs",
				},
			},
			Label: description,
		}
	}

	return nil
}

// GenerateImages generates an images stream from a GitHub repository
func (s *SimpleStreams) GenerateImages(ctx context.Context, owner, repo string) (ss.Products, error) {
	ps := ss.Products{
		ContentID: "images",
		DataType:  "image-downloads",
		Format:    "products:1.0",
		Products:  map[string]ss.Product{},
	}

	rels, err := s.listAllReleases(ctx, owner, repo)
	if err != nil {
		return ps, fmt.Errorf("failed to list releases for %v/%v: %w", owner, repo, err)
	}

	for _, rel := range rels {
		image, err := ParseTag(*rel.TagName)
		if err != nil {
			continue
		}

		if err := s.releaseToProducts(ps.Products, image, rel); err != nil {
			log.WithField("image", image).WithError(err).Warn("Failed to convert release to a product")
		}
	}

	return ps, nil
}

// GetPathURL converts a path (generated by releaseToProducts) and returns the
// GitHub asset URL for the path
func (s *SimpleStreams) GetPathURL(ctx context.Context, owner, repo, name, arch, version, file string) (string, error) {
	image, err := ParseTag(name + "/v" + version)
	if err != nil {
		return "", fmt.Errorf("failed to parse image tag: %v", err)
	}

	if file != "lxd.tar.xz" && file != "rootfs.squashfs" {
		return "", ErrInvalidPath
	}
	switch file {
	case "lxd.tar.xz":
		file = fmt.Sprintf("lxd.%v.tar.xz", arch)
	case "rootfs.squashfs":
		file = fmt.Sprintf("rootfs.%v.squashfs", arch)
	}

	rel, _, err := s.client.Repositories.GetReleaseByTag(ctx, owner, repo, image.String())
	if err != nil {
		return "", fmt.Errorf("failed to retrieve GitHub release: %w", err)
	}

	for _, ass := range rel.Assets {
		if *ass.Name == file {
			return *ass.BrowserDownloadURL, nil
		}
	}

	return "", ErrIncompleteRelease
}
