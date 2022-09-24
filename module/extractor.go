package module

import (
	"fmt"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"

	"github.com/epicbytes/protocommon/common"
)

type dataExtractor struct {
	pgs.Visitor
	pgs.DebuggerCommon
	pgsgo.Context
	extractorData
}

type extractorData struct {
	features map[string]*ModelFeatureCollection
}

func newTagExtractor(d pgs.DebuggerCommon, ctx pgsgo.Context) *dataExtractor {
	v := &dataExtractor{DebuggerCommon: d, Context: ctx}
	v.Visitor = pgs.PassThroughVisitor(v)
	return v
}

func (v *dataExtractor) VisitFile(f pgs.File) (pgs.Visitor, error) {
	var serviceConfig *common.ServiceOption

	_, err := f.Extension(common.E_Service, &serviceConfig)
	if err != nil {
		return nil, err
	}

	//v.Debug(serviceConfig)

	return v, nil
}

func (v *dataExtractor) VisitMessage(f pgs.Message) (pgs.Visitor, error) {
	var findedFeatures *common.ModelFeature
	var parserFeatures *common.ParserOption

	_, err := f.Extension(common.E_ModelFeature, &findedFeatures)
	if err != nil {
		return nil, err
	}

	_, err = f.Extension(common.E_Parser, &parserFeatures)
	if err != nil {
		return nil, err
	}

	msgName := f.Name().String()

	if msgName == "ListEntity" {

		msgName = fmt.Sprintf("%s_%s", f.Parent().Name().String(), f.Name().String())

	}

	if v.features[msgName] == nil {
		v.features[msgName] = &ModelFeatureCollection{}
	}

	if findedFeatures != nil {
		v.features[msgName].features = findedFeatures
	}
	if parserFeatures != nil {
		v.features[msgName].parser = parserFeatures
	}

	if v.features[msgName].features == nil && v.features[msgName].parser == nil {
		return nil, nil
	}

	return v, nil
}

func (v *dataExtractor) VisitMethod(m pgs.Method) (pgs.Visitor, error) {
	var methodOptions *common.MethodOption
	_, err := m.Extension(common.E_Option, &methodOptions)
	if err != nil {
		return nil, err
	}
	//v.Debug(methodOptions)
	return v, nil
}

func (v *dataExtractor) VisitField(f pgs.Field) (pgs.Visitor, error) {
	var tval *common.ModelFieldOption
	ok, err := f.Extension(common.E_FieldOption, &tval)
	if err != nil {
		return nil, err
	}

	msgName := v.Context.Name(f.Message()).String()

	if ok {
		if tval.GetMerged() {
			model := &MergedPickedFieldData{
				Name:      f.Name().String(),
				ProtoType: f.Descriptor().Type.String(),
				Repeated:  f.Type().IsRepeated(),
				Source:    tval.GetSource(),
			}

			if f.Type().IsEmbed() {

				model.Type = f.Type().Embed().Name().String()
			}

			v.features[msgName].fieldsList = append(v.features[msgName].fieldsList, model)
		}
		if tval.GetPicked() {
			model := &MergedPickedFieldData{
				Name:      f.Name().String(),
				ProtoType: f.Descriptor().Type.String(),
				Repeated:  f.Type().IsRepeated(),
				Source:    tval.GetSource(),
			}

			if f.Type().IsEmbed() {

				model.Type = f.Type().Embed().Name().String()
			}
			v.features[msgName].fieldsList = append(v.features[msgName].fieldsList, model)
		}
		if !tval.GetPicked() && !tval.GetMerged() {
			model := &MergedPickedFieldData{
				Name:      f.Name().String(),
				ProtoType: f.Descriptor().Type.String(),
				Repeated:  f.Type().IsRepeated(),
				Source:    tval.GetSource(),
			}
			v.features[msgName].fieldsList = append(v.features[msgName].fieldsList, model)
		}
	}

	return v, nil
}

func (v *dataExtractor) ExtractFeatures(f pgs.File) extractorData {
	v.features = map[string]*ModelFeatureCollection{}
	v.CheckErr(pgs.Walk(v, f))

	return v.extractorData
}
