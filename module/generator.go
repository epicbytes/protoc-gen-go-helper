package module

import (
	"github.com/epicbytes/protocommon/common"
)

type Activity struct {
	Name   string
	Input  string
	Output string
}

type ModelFeatureCollection struct {
	features   *common.ModelFeature
	parser     *common.ParserOption
	fields     *common.ModelFieldOption
	fieldsList []*MergedPickedFieldData
}

type ModelFeatures map[string]*ModelFeatureCollection
