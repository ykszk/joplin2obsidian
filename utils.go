package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	copy2 "github.com/otiai10/copy"
)

func GetTags() (map[string]string, error) {

	tags := make(map[string]string)
	dir, err := ioutil.ReadDir(*SrcPath)
	CheckError(err)

	for _, entry := range dir {
		if entry.IsDir() ||
			path.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := path.Join(*SrcPath, entry.Name())
		tagId, tagName := GetTagInfo(filePath)
		if tagId == "" || tagName == "" {
			continue
		}

		// Either change the # prefix or remove it
		if strings.HasPrefix(tagName, "#") {
			tagName = tagName[1:]
			// tagName = "@" + tagName[1:]
		}

		tags[tagId] = tagName
	}

	return tags, nil
}

func GetTagInfo(filePath string) (tagId string, tagName string) {

	data, err := ioutil.ReadFile(filePath)
	CheckError(err)

	strData := strings.TrimSpace(string(data))
	metaIndex := strings.LastIndex(strData, "\n\n")
	if metaIndex <= 0 {
		return "", ""
	}

	strMeta := strData[metaIndex:]
	strMeta = fmt.Sprintf("%s\n", strMeta)
	r, _ := regexp.Compile("type_: *(.*)\n")
	match := r.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return "", ""
	}

	metaType, err := strconv.Atoi(match[1])
	CheckError(err)
	if metaType != 5 {
		return "", ""
	}

	r, _ = regexp.Compile("id: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return "", ""
	}
	tagId = match[1]

	tagIndex := strings.Index(strData, "\n")
	if tagIndex == -1 {
		return "", ""
	}
	tagName = strData[:tagIndex]

	return tagId, tagName

}

func GetFileInfo(filePath string) (*FileInfo, *string) {
	data, err := ioutil.ReadFile(filePath)
	CheckError(err)
	strData := strings.TrimSpace(string(data))
	metaIndex := strings.LastIndex(strData, "\n\n")
	if metaIndex <= 0 {
    // files with type: _6 tag-note relationship have no content and start with metadata
		r, _ := regexp.Compile("id: *(.*)\n")
		match := r.FindStringSubmatch(strData)
		if len(match) < 2 {
			return nil, nil
		}
		metaIndex = 0
	}

	strMeta := strData[metaIndex:]
	strMeta = fmt.Sprintf("%s\n", strMeta)

	r, _ := regexp.Compile("id: *(.*)\n")
	match := r.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return nil, nil
	}
	metaId := match[1]

	r, _ = regexp.Compile("type_: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) < 2 {
		return nil, nil
	}
	metaType, err := strconv.Atoi(match[1])
	CheckError(err)
	if 1 != metaType && 2 != metaType && 4 != metaType && metaType != 6 {
		return nil, nil
	}

	metaParentId := ""
	r, _ = regexp.Compile("parent_id: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaParentId = match[1]
	}

	metaFileExt := ""
	r, _ = regexp.Compile("file_extension: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		metaFileExt = match[1]
	}

	var sourceURL string
	r, _ = regexp.Compile("source_url: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		sourceURL = match[1]
	}

	var creationTime time.Time
	r, _ = regexp.Compile("user_created_time: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		t, _ := time.Parse(time.RFC3339, match[1])
		creationTime = t
	}

	var updatedTime time.Time
	r, _ = regexp.Compile("user_updated_time: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		t, _ := time.Parse(time.RFC3339, match[1])
		updatedTime = t
	}

	var tagId string
	r, _ = regexp.Compile("tag_id: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		tagId = match[1]
	}

	var tagNoteId string
	r, _ = regexp.Compile("note_id: *(.*)\n")
	match = r.FindStringSubmatch(strMeta)
	if len(match) >= 2 {
		tagNoteId = match[1]
	}

	r, _ = regexp.Compile("(.*)\n")
	match = r.FindStringSubmatch(strData)
	if len(match) < 2 {
		return nil, nil
	}
	name := strings.TrimSpace(match[1])

	return &FileInfo{
		name:         name,
		metaIndex:    metaIndex,
		metaId:       metaId,
		metaType:     metaType,
		metaParentId: metaParentId,
		metaFileExt:  metaFileExt,
		creationTime: creationTime,
		updatedTime:  updatedTime,
		sourceURL:    sourceURL,
		tagId:        tagId,
		tagNoteId:    tagNoteId,
	}, &strData
}

var StepDesc = [6]string{
	"Initializing",
	"Extracting Metadata", //1
	"Adding metadata to Articles",
	"Rebuilding Folders",
	"Rebuilding Articles",
	"Saving Data",
}

func HandlingCoreBusiness(progress chan<- int, done chan<- bool) {
	folderMap := make(map[string]*Folder)
	articleMap := make(map[string]*Article)
	resMap := make(map[string]*Resource)
	articleTagsMap := make(map[string][]string)

	tagMap, err := GetTags()
	CheckError(err)

	c, err := ioutil.ReadDir(*SrcPath)
	CheckError(err)
	for _, entry := range c {
		if entry.IsDir() ||
			path.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := path.Join(*SrcPath, entry.Name())
		fi, rawData := GetFileInfo(filePath)
		if fi == nil {
			continue
		}

		switch metaType := fi.metaType; metaType {
		case 1:
			content := (*rawData)[:fi.metaIndex]
			r, _ := regexp.Compile("(.*\n)")
			match := r.FindStringIndex(content)
			if len(match) == 2 {
				content = strings.TrimSpace(content[match[1]:])
			}
			article := Article{FileInfo: fi, content: content, sourceURL: fi.sourceURL}
			articleMap[article.metaId] = &article
		case 2:
			folder := Folder{FileInfo: fi}
			folderMap[folder.metaId] = &folder
		case 4:
			resMap[fi.metaId] = &Resource{FileInfo: fi}
		case 6:
			if tags, ok := articleTagsMap[fi.tagNoteId]; ok {
				articleTagsMap[fi.tagNoteId] = append(tags, tagMap[fi.tagId])
			} else {
				articleTagsMap[fi.tagNoteId] = []string{tagMap[fi.tagId]}
			}
		}

		progress <- 1
	}

	AddMetadataToArticles(&articleMap, &articleTagsMap, progress)
	RebuildFoldersRelationship(&folderMap, progress)
	RebuildArticlesRelationship(&articleMap, &folderMap, progress)

	err = copy2.Copy(path.Join(*SrcPath, ResourcesFolder), path.Join(*DestPath, ResourcesFolder))
	CheckError(err)

	for _, article := range articleMap {
		FixResourceRef(article, &resMap, &articleMap)
		article.save()
		progress <- 5
	}

	close(progress)
	done <- true
}

func FixResourceRef(article *Article, resMap *map[string]*Resource, articleMap *map[string]*Article) {
	content := article.content
	r, _ := regexp.Compile(`(!?)\[(.*?)]\(:/(.*?)\)`)
	matchAll := r.FindAllStringSubmatchIndex(content, -1)
	for i := len(matchAll) - 1; i >= 0; i-- {
		match := matchAll[i]
		resId := strings.Split(content[match[6]:match[7]], " ")[0]
		resId_wo_heading := strings.Split(resId, "#")[0]

		var resFileName string
		if res, prs := (*resMap)[resId]; prs {
			resFileName = res.getFileName()
		} else if res, prs := (*articleMap)[resId]; prs {
			resFileName = path.Join(res.folder.getRelativePath(), res.getValidName())
		} else if res, prs := (*articleMap)[resId_wo_heading]; prs {
			resFileName = path.Join(res.folder.getRelativePath(), res.getValidName()) + "#" + strings.Split(resId, "#")[1]
		} else {
			resFileName = path.Join("resources", resId) // help to find lost resource
		}
		content = fmt.Sprintf("%s[[%s]]%s", content[:match[3]], resFileName, content[match[1]:])
	}
	article.content = content
}

func _FixResourceRef_html(article *Article, resMap *map[string]*Resource, articleMap *map[string]*Article) {
	content := article.content
	r, _ := regexp.Compile(`src=":/([0-9a-f]{32})"`)
	matchAll := r.FindAllStringSubmatchIndex(content, -1)
	for i := len(matchAll) - 1; i >= 0; i-- {
		match := matchAll[i]
		resId := content[match[2]:match[3]]

		var resFileName string
		if res, prs := (*resMap)[resId]; prs {
			resFileName = res.getFileName()
		} else if res, prs := (*articleMap)[resId]; prs {
			resFileName = path.Join(res.folder.getRelativePath(), res.getValidName())
		} else {
			resFileName = path.Join("resources", resId) // help to find lost resource
		}
		content = fmt.Sprintf(`%ssrc="%s/%s"%s`, content[:match[0]], DstResourcesFolder, resFileName, content[match[1]:])
	}
	article.content = content
}


func FixResourceRef(article *Article, resMap *map[string]*Resource, articleMap *map[string]*Article) {
	_FixResourceRef_md(article, resMap, articleMap)
	_FixResourceRef_html(article, resMap, articleMap)
}

func AddMetadataToArticles(articleMap *map[string]*Article, articleTagsMap *map[string][]string, progress chan<- int) {
	for noteId, article := range *articleMap {
		var (
			metadata         string
			tagsSection      string
			sourceUrlSection string
		)
		if tags, ok := (*articleTagsMap)[noteId]; ok {
			tagsSection = fmt.Sprintf("tags: %s\n", strings.Join(tags, ", "))
		}

		if article.sourceURL != "" {
			sourceUrlSection = fmt.Sprintf("source_url: %s\n", article.sourceURL)
		}

		if tagsSection != "" || sourceUrlSection != "" {
			metadata = fmt.Sprintf("---\n%s%s---", tagsSection, sourceUrlSection)
		}

		if metadata != "" {
			article.content = fmt.Sprintf("%s\n\n%s", metadata, article.content)
		}

		progress <- 2
	}
}

func RebuildFoldersRelationship(folderMap *map[string]*Folder, progress chan<- int) {
	for _, folder := range *folderMap {
		if len(folder.metaParentId) == 0 {
			continue
		}
		parent := (*folderMap)[folder.metaParentId]
		folder.parent = parent
		progress <- 3
	}
}

func RebuildArticlesRelationship(articleMap *map[string]*Article, folderMap *map[string]*Folder, progress chan<- int) {
	for _, article := range *articleMap {
		if len(article.metaParentId) == 0 {
			continue
		}
		parent := (*folderMap)[article.metaParentId]
		article.folder = parent
		progress <- 4
	}
}
