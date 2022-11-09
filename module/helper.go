package module

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	. "github.com/dave/jennifer/jen"
	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
)

type mod struct {
	*pgs.ModuleBase
	pgsgo.Context
	ModuleName string
}

func NewHelper(moduleName string) pgs.Module {
	return &mod{ModuleName: moduleName, ModuleBase: &pgs.ModuleBase{}}
}

func (m *mod) InitContext(c pgs.BuildContext) {
	m.ModuleBase.InitContext(c)
	m.Context = pgsgo.InitContext(c.Parameters())
}

func (mod) Name() string {
	return "helpers"
}

func (m mod) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {

	module := m.Parameters().Str("module")
	extractor := newTagExtractor(m, m.Context)

	for _, f := range targets {
		extrData := extractor.ExtractFeatures(f)
		helpersName := m.Context.OutputPath(f).SetExt(".helpers.go").String()
		outdir := m.Parameters().Str("outdir")
		helpersFilename := helpersName
		if outdir != "" {
			helpersFilename = filepath.Join(outdir, helpersName)
		}

		if module != "" {
			helpersFilename = strings.ReplaceAll(helpersFilename, string(filepath.Separator), "/")
			trim := module + "/"
			if !strings.HasPrefix(helpersFilename, trim) {
				m.Debug(fmt.Sprintf("%v: generated file does not match prefix %q", helpersFilename, module))
				m.Exit(1)
			}
			helpersFilename = strings.TrimPrefix(helpersFilename, trim)
		}

		file := NewFile("pb")

		pathToKeeper := fmt.Sprintf("%s/internal/keeper", m.ModuleName)
		file.ImportName(pathToKeeper, "keeper")
		pathToCommon := "github.com/epicbytes/protocommon/common"
		file.ImportName(pathToCommon, "common")
		pathToJson := "github.com/goccy/go-json"
		file.ImportName(pathToJson, "json")
		pathToBson := "go.mongodb.org/mongo-driver/bson"
		file.ImportName(pathToBson, "bson")
		pathToPrimitive := "go.mongodb.org/mongo-driver/bson/primitive"
		file.ImportName(pathToPrimitive, "primitive")
		pathToOptions := "go.mongodb.org/mongo-driver/mongo/options"
		file.ImportName(pathToOptions, "options")
		pathToFiber := "github.com/gofiber/fiber/v2"
		file.ImportName(pathToFiber, "fiber")
		pathToDeepCopy := "github.com/barkimedes/go-deepcopy"
		file.ImportName(pathToDeepCopy, "deepcopy")

		keys := make([]string, 0, len(extrData.features))
		for k := range extrData.features {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if extrData.features != nil {
			for _, modelName := range keys {
				feature := extrData.features[modelName]
				//m.Debug("MODELNAME", modelName)
				// Add helpers for Keeper
				if feature.features != nil {
					file.Func().Params(Id("x").Op("*").Id(modelName)).Id("EncryptFields").Params(
						Id("ctx").Qual("context", "Context"),
						Id("keepr").Qual(pathToKeeper, "Keeper"),
					).Params(Op("*").Id(modelName)).BlockFunc(func(group *Group) {
						group.List(Id("str"), Err()).Op(":=").Qual(pathToDeepCopy, "Anything").Call(Id("x"))
						group.If(Err().Op("!=").Nil()).Block(
							Qual("fmt", "Println").Call(Err()),
						)
						group.Id("keepr").Dot("TransitEncrypt").Call(Id("ctx"), Id("str"), Lit(feature.features.KeeperKey))
						group.Return(Id("str").Assert(Op("*").Id(modelName)))
					})
					file.Func().Params(Id("x").Op("*").Id(modelName)).Id("DecryptFields").Params(
						Id("ctx").Qual("context", "Context"),
						Id("keepr").Qual(pathToKeeper, "Keeper"),
					).Params(Op("*").Id(modelName)).BlockFunc(func(group *Group) {
						group.List(Id("str"), Err()).Op(":=").Qual(pathToDeepCopy, "Anything").Call(Id("x"))
						group.If(Err().Op("!=").Nil()).Block(
							Qual("fmt", "Println").Call(Err()),
						)
						group.Id("keepr").Dot("TransitDecrypt").Call(Id("ctx"), Id("str"), Lit(feature.features.KeeperKey))
						group.Return(Id("str").Assert(Op("*").Id(modelName)))
						/*
							str, err := deepcopy.Anything(x)
								if err != nil {
									fmt.Println(err)
								}
								keepr.TransitDecrypt(ctx, str, "providers_token")
								return str.(*MerchantEntity)
						*/
					})
				}
				// Parser features
				if feature.parser != nil {
					if feature.parser.GetSwag() {
						file.Comment(fmt.Sprintf("swagger:parameters %sWrapper", Camel(modelName)))
						file.Comment(fmt.Sprintf("%sWrapper wrapper for %s", modelName, modelName))
						file.Type().Id(fmt.Sprintf("%sWrapper", modelName)).Struct(
							Comment("In: body"),
							Id("Body").Id(modelName),
						)
					}
					if feature.parser.GetPaging() && strings.HasSuffix(modelName, "Request") {
						file.Func().Params(Id("x").Op("*").Id(modelName)).Id("GetFilter").Params().Params(Qual(pathToBson, "M")).BlockFunc(func(group *Group) {
							group.Id("query").Op(":=").Qual(pathToBson, "M").Values()
							for _, field := range feature.fieldsList {
								if field.Name == "skip" || field.Name == "limit" {
									continue
								}
								switch field.ProtoType {
								case "TYPE_UINT32":
									group.If(Id("x").Dot(Pascal(field.Name)).Op(">").Lit(0)).BlockFunc(func(group2 *Group) {
										group2.Id("query").Index(Lit(field.Name)).Op("=").Id("x").Dot(Pascal(field.Name))
									})
								case "TYPE_STRING":
									group.If(Len(Id("x").Dot(Pascal(field.Name))).Op(">").Lit(0)).BlockFunc(func(group2 *Group) {
										group2.Id("query").Index(Lit(field.Name)).Op("=").Qual(pathToPrimitive, "Regex").Values(DictFunc(func(dict Dict) {
											dict[Id("Pattern")] = Id("x").Dot(Pascal(field.Name))
											dict[Id("Options")] = Lit("")
										}))
									})
								default:
								}
								m.Debug(modelName, field.Name, field.Type)
							}

							group.Return(Id("query"))
						})

						file.Func().Params(Id("x").Op("*").Id(modelName)).Id("GetOptions").Params().Op("*").Qual(pathToOptions, "FindOptions").BlockFunc(func(group *Group) {
							group.Var().Id("opts").Id("=").Op("&").Qual(pathToOptions, "FindOptions").Values()
							if feature.parser.GetList() {
								group.Var().Id("limit").Int64().Op("=").Lit(20)
								group.If(Id("x").Dot("Limit").Op(">").Lit(0)).Block(
									Id("limit").Op("=").Id("x").Dot("Limit"),
								)
								group.Id("opts").Dot("SetLimit").Call(Id("limit"))
								group.Id("opts").Dot("SetSkip").Call(Id("x").Dot("Skip"))
								group.Id("opts").Dot("SetSort").Call(Qual(pathToBson, "M").Values(Dict{Lit("_id"): Lit(1)}))
							}
							group.Return(Id("opts"))
						})
					}
					if feature.parser.GetMerge() || feature.parser.GetMergeFrom() != "" {
						file.Func().Params(
							Id("x").Op("*").Id(feature.parser.GetMergeFrom()),
						).Id(fmt.Sprintf("MergeFrom%s", modelName)).Params(
							Id("request").Op("*").Id(modelName),
						).BlockFunc(func(group *Group) {
							group.If(Id("x").Op("==").Nil()).Block(Return())

							for _, field := range feature.fieldsList {
								if !field.Merged {
									continue
								}
								group.Id("x").Dot(Pascal(field.Name)).Op("=").Id("request").Dot(fmt.Sprintf("Get%s()", Pascal(field.Name)))
							}
						})
					}
					if feature.parser.GetPick() {
						file.Func().Params(
							Id("x").Op("*").Id(modelName),
						).Id(fmt.Sprintf("PickFrom%s", feature.parser.PickWith)).ParamsFunc(func(group *Group) {
							if feature.parser.GetList() {
								group.Id("request").Index().Op("*").Id(feature.parser.PickWith)
								if feature.parser.GetPaging() {
									group.Id("pagination").Op("*").Qual(pathToCommon, "Pagination")
								}
							} else {
								group.Id("request").Op("*").Id(feature.parser.PickWith)
							}
						}).BlockFunc(func(group *Group) {
							group.If(Id("request").Op("==").Nil()).Block(Return())
							if feature.parser.GetList() {
								group.Var().Id("items").Op("=").Make(Index().Op("*").Id(fmt.Sprintf("%s_ListEntity", modelName)), Lit(0))
								group.If(Len(Id("request")).Op(">").Lit(0)).Block(
									For(List(Id("_"), Id("req")).Op(":=").Range().Id("request")).BlockFunc(func(groupFunc *Group) {
										groupFunc.Var().Id("item").Op("=").New(Id(fmt.Sprintf("%s_ListEntity", modelName)))
										if extrData.features[fmt.Sprintf("%s_ListEntity", modelName)] != nil {
											for _, field := range extrData.features[fmt.Sprintf("%s_ListEntity", modelName)].fieldsList {
												if !field.Picked {
													continue
												}
												groupFunc.Id("item").Dot(Pascal(field.Name)).Op("=").Id("req").Dot(fmt.Sprintf("Get%s()", Pascal(field.Name)))
											}
										}
										groupFunc.Id("items").Op("=").Append(Id("items"), Id("item"))
									}),
								)
								group.Id("x").Dot("Items").Op("=").Id("items")
								if feature.parser.GetPaging() {
									group.Id("x").Dot("Pagination").Op("=").Id("pagination")
								}
							} else {
								for _, field := range feature.fieldsList {
									if !field.Picked {
										continue
									}
									group.Id("x").Dot(Pascal(field.Name)).Op("=").Id("request").Dot(fmt.Sprintf("Get%s()", Pascal(field.Name)))
								}
							}
						})
					}
					if feature.parser.GetFiber() {
						m.Debug(modelName, feature)
						file.Func().Params(
							Id("x").Op("*").Id(modelName),
						).Id("BindFromFiber").Params(
							Id("ctx").Op("*").Qual(pathToFiber, "Ctx"),
						).Error().BlockFunc(func(group *Group) {
							group.Var().Err().Error()
							group.Err().Op("=").Id("ctx").Dot("QueryParser").Call(Id("x"))
							group.If(Err().Op("!=").Nil()).Block(
								Return(Err()),
							)

							hasBodyFields := false
							for _, field := range feature.fieldsList {
								if field.Source == "body" {
									hasBodyFields = true
								}
							}
							if hasBodyFields {
								group.Err().Op("=").Id("ctx").Dot("BodyParser").Call(Id("x"))
								group.If(Err().Op("!=").Nil()).Block(
									Return(Err()),
								)
							}

							for _, field := range feature.fieldsList {
								if field.Source == "path" {
									group.List(Id(field.Name), Err()).Op(":=").Id("ctx").Dot("ParamsInt").Call(Lit(field.Name))
									group.If(Err().Op("!=").Nil()).Block(
										Return(Err()),
									)
									group.Id("x").Dot(Pascal(field.Name)).Op("=").Uint32().Call(Id(field.Name))

								}
								if field.Source == "context" {
									group.If(Id("ctx").Dot("Locals").Call(Lit(field.Name))).Op("!=").Nil().BlockFunc(func(group *Group) {
										group.Id("x").Dot(Pascal(field.Name)).Op("=").Id("ctx").Dot("Locals").Call(Lit(field.Name)).Assert(Id(ProtoTypesMap[field.ProtoType]))
									})
								}
							}

							group.Return(Nil())
						})
					}
				}

				// Common helpers for each one messages
				file.Func().Params(Id("x").Op("*").Id(modelName)).Id("MustMarshalBinary").Params().Params(Index().Byte()).BlockFunc(func(group *Group) {
					group.List(Id("b"), Err()).Op(":=").Qual(pathToJson, "Marshal").Call(Id("x"))
					group.If(Err().Op("!=").Nil()).Block(
						Qual("fmt", "Println").Call(Err()),
					)
					group.Return(Id("b"))
				})
				file.Func().Params(Id("x").Op("*").Id(modelName)).Id("UnmarshalBinary").Params(Id("data").Index().Byte()).Error().BlockFunc(func(group *Group) {
					group.If(Err().Op(":=").Qual(pathToJson, "Unmarshal").Call(
						Id("data"),
						Id("x"),
					).Op(";").Err().Op("!=").Nil()).Block(
						Return(Err()),
					)
					group.Return(Nil())
				})
			}
		}

		err := file.Save(helpersFilename)
		if err != nil {
			m.Debug(GetErrWithLinesNumber(err))
			return nil
		}
	}
	return m.Artifacts()
}
