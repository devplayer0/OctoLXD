package simplestreams

import "errors"

var (
	// ErrInvalidTag indicates a tag isn't a valid image + version combo
	ErrInvalidTag = errors.New("invalid tag format")
	// ErrMetadataMissing indicates the metadata.yaml file is missing
	ErrMetadataMissing = errors.New("metadata.yaml missing from lxd.tar.xz")
	// ErrIncompleteRelease indicates a release is missing key assets
	ErrIncompleteRelease = errors.New("release is missing required assets")
)
