package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
  "time"
)

var SrcPath *string
var DestPath *string
var AddTag *bool
var AddSourceUrl *bool

const SrcResourcesFolder string = "resources"
var DstResourcesFolder *string

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

type FileInfo struct {
	name         string
	metaIndex    int
	metaId       string
	metaParentId string
	metaType     int //1:Article 2:Folder 4:Resource 5:Tag 6:Article-Tag mapping
	metaFileExt  string
	creationTime time.Time
	updatedTime  time.Time
	sourceURL    string
	tagId        string
	tagNoteId    string
}

func (fi FileInfo) getValidName() string {
	r := strings.NewReplacer(
		"*", ".",
		"\"", "''",
		"\\", "-",
		"/", "_",
		"<", ",",
		">", ".",
		":", ";",
		"|", "-",
		"?", "!")
	return r.Replace(fi.name)
}

type Folder struct {
	*FileInfo
	parent *Folder
}

func (f Folder) getPath() string {
	return path.Join(*DestPath, f.getRelativePath())
}

func (f Folder) getRelativePath() string {
	if f.parent == nil {
		return f.getValidName()
	} else {
		return path.Join(f.parent.getRelativePath(), f.getValidName())
	}
}

type Article struct {
	*FileInfo
	folder    *Folder
	content   string
	sourceURL string
}

func (a Article) getPath() string {
	return fmt.Sprintf("%s.md", path.Join(a.folder.getPath(), a.getValidName()))
}
func (a Article) save() {
	filePath := a.getPath()
	dirName := path.Dir(filePath)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.MkdirAll(dirName, 0755)
		CheckError(err)
	}
	err := ioutil.WriteFile(filePath, []byte(a.content), 0644)
	CheckError(err)

	// Use article creation and modify time
	err = os.Chtimes(filePath, a.creationTime, a.updatedTime)
	CheckError(err)
}

type Resource struct {
	*FileInfo
}

func (r Resource) getFileName() string {
	var fileName string
	if /*len(r.metaFileExt) > 0*/ false {
		fileName = fmt.Sprintf("%s.%s", r.metaId, r.metaFileExt)
	} else {
		resPath := path.Join(*SrcPath, "resources")
		c, err := ioutil.ReadDir(resPath)
		CheckError(err)
		for _, entry := range c {
			if entry.IsDir() {
				continue
			}
			if strings.Index(entry.Name(), r.metaId) >= 0 {
				fileName = entry.Name()
				break
			}
		}
	}
	return fileName
}
