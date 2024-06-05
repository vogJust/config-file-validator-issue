package finder

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"fmt"

	"github.com/Boeing/config-file-validator/pkg/filetype"
	"github.com/Boeing/config-file-validator/pkg/misc"
)

type FileSystemFinder struct {
	PathRoots        []string
	FileTypes        []filetype.FileType
	ExcludeDirs      map[string]struct{}
	ExcludeFileTypes map[string]struct{}
	Depth            *int
	AdditionalFiles  map[string]string
}

type FSFinderOptions func(*FileSystemFinder)

// Set the CLI SearchPath
func WithPathRoots(paths ...string) FSFinderOptions {
	return func(fsf *FileSystemFinder) {
		fsf.PathRoots = paths
	}
}

// Add a custom list of file types to the FSFinder
func WithFileTypes(fileTypes []filetype.FileType) FSFinderOptions {
	return func(fsf *FileSystemFinder) {
		fsf.FileTypes = fileTypes
	}
}

// Add a custom list of file types to the FSFinder
func WithExcludeDirs(excludeDirs []string) FSFinderOptions {
	return func(fsf *FileSystemFinder) {
		fsf.ExcludeDirs = misc.ArrToMap(excludeDirs...)
	}
}

// WithExcludeFileTypes adds excluded file types to FSFinder.
func WithExcludeFileTypes(types []string) FSFinderOptions {
	return func(fsf *FileSystemFinder) {
		fsf.ExcludeFileTypes = misc.ArrToMap(types...)
	}
}

// WithDepth adds the depth for search recursion to FSFinder
func WithDepth(depthVal int) FSFinderOptions {
	return func(fsf *FileSystemFinder) {
		fsf.Depth = &depthVal
	}
}

// AddFiles adds additional files with specified formats to FSFinder
func AddFiles(files []string) FSFinderOptions {
	return func(fsf *FileSystemFinder) {
		if len(files) > 0 {
		fsf.AdditionalFiles= make(map[string]string)
			for i := range files {
				kv := strings.Split(files[i], ":")
				if len(kv) == 2 {
					fsf.AdditionalFiles[kv[0]] = kv[1]
				}
			}
		}
	}
}

func readConfig(file string) {
	if len(file) > 0 {
		//WORK IN PROGRESS
	}
}

func FileSystemFinderInit(opts ...FSFinderOptions) *FileSystemFinder {
	defaultExcludeDirs := make(map[string]struct{})
	defaultPathRoots := []string{"."}

	fsfinder := &FileSystemFinder{
		PathRoots:   defaultPathRoots,
		FileTypes:   filetype.FileTypes,
		ExcludeDirs: defaultExcludeDirs,
	}

	for _, opt := range opts {
		opt(fsfinder)
	}

	return fsfinder
}

// Find implements the FileFinder interface by calling findOne on
// all the PathRoots and providing the aggregated FileMetadata after
// ignoring all the duplicate files
func (fsf FileSystemFinder) Find() ([]FileMetadata, error) {
	seen := make(map[string]struct{}, 0)
	uniqueMatches := make([]FileMetadata, 0)
	for _, pathRoot := range fsf.PathRoots {
		matches, err := fsf.findOne(pathRoot)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			absPath, err := filepath.Abs(match.Path)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[absPath]; ok {
				continue
			}
			uniqueMatches = append(uniqueMatches, match)
			seen[absPath] = struct{}{}
		}
	}
	return uniqueMatches, nil
}

// findOne recursively walks through all subdirectories (excluding the excluded subdirectories)
// and identifying if the file matches a type defined in the fileTypes array for a
// single path and returns the file metadata.
func (fsf FileSystemFinder) findOne(pathRoot string) ([]FileMetadata, error) {
	var matchingFiles []FileMetadata

	// check that the path exists before walking it or the error returned
	// from filepath.Walk will be very confusing and undescriptive
	if _, err := os.Stat(pathRoot); os.IsNotExist(err) {
		return nil, err
	}

	var depth int
	if fsf.Depth != nil {
		depth = *fsf.Depth
	}

	maxDepth := strings.Count(pathRoot, string(os.PathSeparator)) + depth

	err := filepath.WalkDir(pathRoot,
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// determine if directory is in the excludeDirs list or if the depth is greater than the maxDepth
			_, isExcluded := fsf.ExcludeDirs[dirEntry.Name()]
			if dirEntry.IsDir() && ((fsf.Depth != nil && strings.Count(path, string(os.PathSeparator)) > maxDepth) || isExcluded) {
				return filepath.SkipDir
			}

			if !dirEntry.IsDir() {
				// filepath.Ext() returns the extension name with a dot so it
				// needs to be removed.

				walkFileExtension := strings.TrimPrefix(filepath.Ext(path), ".")

				// Checking for file name matching with additional files
				// Check matching base name, return vals with passed KVPair value as fileType
				//if _, ok := fsf.AdditionalFiles[dirEntry.Name()]; ok {
				//	extensionLowerCase := AdditionalFiles[dirEntry.Name()]
				//}

				if len(walkFileExtension) == 0 {
					var ok bool
					walkFileExtension, ok = fsf.AdditionalFiles[dirEntry.Name()]
					if !ok {
						walkFileExtension = ""
					}
				}

				fmt.Println(path)
				fmt.Println(walkFileExtension)

				// Checking for case sensitive exclusion
				if _, ok := fsf.ExcludeFileTypes[walkFileExtension]; ok {
					fmt.Println("excluded")
					return nil
				}
				extensionLowerCase := strings.ToLower(walkFileExtension)

				// Check each fileType, ignore non-matching extensions
				for _, fileType := range fsf.FileTypes {
					if _, ok := fileType.Extensions[extensionLowerCase]; ok {
						fileMetadata := FileMetadata{dirEntry.Name(), path, fileType}
						matchingFiles = append(matchingFiles, fileMetadata)
						break
					}
				}
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return matchingFiles, nil
}
