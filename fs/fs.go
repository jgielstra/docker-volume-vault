package fs

import (
	"log"
	"strings"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hashicorp/vault/api"
	"golang.org/x/sys/unix"
)

const (
	EISDIR = fuse.Status(unix.EISDIR)
)

type fs struct {
	pathfs.FileSystem
	client *api.Client
}

// NewKeywhizFs readies a KeywhizFs struct and its parent filesystem objects.
func NewFs(client *api.Client) (*fs, nodefs.Node) {
	defaultfs := pathfs.NewDefaultFileSystem()            // Returns ENOSYS by default
	readonlyfs := pathfs.NewReadonlyFileSystem(defaultfs) // R/W calls return EPERM

	kwfs := &fs{readonlyfs, client}
	nfs := pathfs.NewPathNodeFs(kwfs, nil)
	//nfs.SetDebug(true)
	return kwfs, nfs.Root()
}

func GetDirectories(keys []interface{}) map[string]bool {
	res := map[string]bool{}
	for _, k := range keys {
		k := k.(string)
		res[strings.TrimRight(k, "/")] = true
	}
	return res
}

// GetAttr is a FUSE function which tells FUSE which files and directories exist.
func (f *fs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	// TODO multiple rest calls,  perhaps cache atts/paths by mount point ??
	switch {
	case name == "":
		return f.directoryAttr(1, 0755), fuse.OK
	case name == "sys":
		return f.directoryAttr(1, 0755), fuse.OK
	case name == "secret":
		return f.directoryAttr(1, 0755), fuse.OK

	case strings.HasPrefix(name, "secret/"):

		// IF we have  an attribute, we know it is a folder..
		atts, err := f.client.Logical().Read(name)
		if err != nil {
			log.Println("[ERR]: %v", err)
			return nil, fuse.ENOENT
		}
		if atts != nil && len(atts.Data) > 0 {
			return f.directoryAttr(1, 0755), fuse.OK
		}
		// IF we have  child node, we know it is a folder..
		dirs, err := f.client.Logical().List(name)
		if err != nil {
			log.Println("[ERR]: %v", err)
			return nil, fuse.ENOENT
		}
		if dirs != nil && len(dirs.Data) > 0 {
			return f.directoryAttr(1, 0755), fuse.OK
		}
	}
	return f.secretAttr("string for length"), fuse.OK
}

// Open is a FUSE function where an in-memory open file struct is constructed.
func (f *fs) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	var file nodefs.File
	switch {
	case name == "" || name == "secret" || name == "sys":
		return nil, EISDIR
	case strings.HasPrefix(name, "secret/"):
		lslash := strings.LastIndex(name, "/")
		attr := name[lslash+1 : len(name)]
		path := name[0:lslash]
		s, err := f.client.Logical().Read(path)
		if err != nil {
			log.Println(err)
			return nil, fuse.ENOENT
		}
		if s != nil && s.Data != nil {
			if cont, ok := s.Data[attr]; ok {
				file = nodefs.NewReadOnlyFile(nodefs.NewDataFile([]byte(cont.(string) + "\n")))
				return file, fuse.OK
			}
		}
	}
	return nil, fuse.ENOENT
}

// OpenDir is a FUSE function called when performing a directory listing.
func (f *fs) OpenDir(name string, context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	var entries []fuse.DirEntry
	switch {
	case name == "":
		mounts, err := f.client.Sys().ListMounts()
		if err != nil || len(mounts) == 0 {
			log.Printf("[ERR] Opendir: %v", err)
			return entries, fuse.OK
		}

		entries = make([]fuse.DirEntry, 0, len(mounts))
		for name, _ := range mounts {
			entries = append(entries, fuse.DirEntry{
				Mode: unix.S_IFDIR,
				Name: strings.TrimSuffix(name, "/"),
			})
		}
	case strings.HasPrefix(name, "secret"):
		listing, err := f.client.Logical().List(name)
		if err != nil {
			log.Println(err)
			return entries, fuse.OK
		}
		if listing != nil {
			if _, ok := listing.Data["keys"]; ok {
				keys := GetDirectories(listing.Data["keys"].([]interface{}))
				for name, _ := range keys {
					entries = append(entries, fuse.DirEntry{
						Mode: unix.S_IFDIR,
						Name: name, //strings.TrimSuffix(name, "/"),
					})
				}
			}
		}
		files, err := f.client.Logical().Read(name)
		if err != nil {
			log.Println(err)
			return entries, fuse.OK
		}
		if files != nil && files.Data != nil {
			for k, _ := range files.Data {
				entries = append(entries, fuse.DirEntry{
					Mode: unix.S_IFREG,
					Name: k, //strings.TrimSuffix(name, "/"),
				})
			}
		}
	}
	return entries, fuse.OK
}

// Unlink is a FUSE function called when an object is deleted.
func (f *fs) Unlink(name string, context *fuse.Context) fuse.Status {
	return fuse.EACCES
}

// secretAttr constructs a fuse.Attr based on a given Secret.
func (f *fs) secretAttr(s string) *fuse.Attr {
	size := uint64(len(s))
	attr := &fuse.Attr{
		Size: size,
		Mode: 0444 | unix.S_IFREG,
	}
	return attr
}

// directoryAttr constructs a generic directory fuse.Attr with the given parameters.
func (f *fs) directoryAttr(subdirCount int, mode uint32) *fuse.Attr {
	attr := &fuse.Attr{
		Size: uint64(subdirCount),
		Mode: fuse.S_IFDIR | mode,
	}
	return attr
}
