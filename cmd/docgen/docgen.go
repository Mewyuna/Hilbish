package main

import (
	"fmt"
	"path/filepath"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"os"
	"sync"
)

var header = `---
title: %s %s
description: %s
layout: doc
menu:
  docs:
    parent: "API"
---

`

type emmyPiece struct {
	DocPiece *docPiece
	Annotations []string
	Params []string // we only need to know param name to put in function
	FuncName string
}

type module struct {
	Types []docPiece
	Docs []docPiece
	Fields []docPiece
	Properties []docPiece
	ShortDescription string
	Description string
	ParentModule string
	HasInterfaces bool
	HasTypes bool
}

type docPiece struct {
	Doc []string
	FuncSig string
	FuncName string
	Interfacing string
	ParentModule string
	GoFuncName string
	IsInterface bool
	IsMember bool
	IsType bool
	Fields []docPiece
	Properties []docPiece
}

type tag struct {
	id string
	fields []string
}

var docs = make(map[string]module)
var interfaceDocs = make(map[string]module)
var emmyDocs = make(map[string][]emmyPiece)
var typeTable = make(map[string][]string) // [0] = parentMod, [1] = interfaces
var prefix = map[string]string{
	"main": "hl",
	"hilbish": "hl",
	"fs": "f",
	"commander": "c",
	"bait": "b",
	"terminal": "term",
}

func getTagsAndDocs(docs string) (map[string][]tag, []string) {
	pts := strings.Split(docs, "\n")
	parts := []string{}
	tags := make(map[string][]tag)

	for _, part := range pts {
		if strings.HasPrefix(part, "#") {
			tagParts := strings.Split(strings.TrimPrefix(part, "#"), " ")
			if tags[tagParts[0]] == nil {
				var id string
				if len(tagParts) > 1 {
					id = tagParts[1]
				}
				tags[tagParts[0]] = []tag{
					{id: id},
				}
				if len(tagParts) >= 2 {
					tags[tagParts[0]][0].fields = tagParts[2:]
				}
			} else {
				fleds := []string{}
				if len(tagParts) >= 2 {
					fleds = tagParts[2:]
				}
				tags[tagParts[0]] = append(tags[tagParts[0]], tag{
					id: tagParts[1],
					fields: fleds,
				})
			}
		} else {
			parts = append(parts, part)
		}
	}

	return tags, parts
}

func docPieceTag(tagName string, tags map[string][]tag) []docPiece {
	dps := []docPiece{}
	for _, tag := range tags[tagName] {
		dps = append(dps, docPiece{
			FuncName: tag.id,
			Doc: tag.fields,
		})
	}

	return dps
}

func setupDocType(mod string, typ *doc.Type) *docPiece {
	docs := strings.TrimSpace(typ.Doc)
	tags, doc := getTagsAndDocs(docs)

	if tags["type"] == nil {
		return nil
	}
	inInterface := tags["interface"] != nil

	var interfaces string
	typeName := strings.ToUpper(string(typ.Name[0])) + typ.Name[1:]
	typeDoc := []string{}

	if inInterface {
		interfaces = tags["interface"][0].id
	}

	fields := docPieceTag("field", tags)
	properties := docPieceTag("property", tags)

	for _, d := range doc {
		if strings.HasPrefix(d, "---") {
			// TODO: document types in lua
			/*
			emmyLine := strings.TrimSpace(strings.TrimPrefix(d, "---"))
			emmyLinePieces := strings.Split(emmyLine, " ")
			emmyType := emmyLinePieces[0]
			if emmyType == "@param" {
				em.Params = append(em.Params, emmyLinePieces[1])
			}
			if emmyType == "@vararg" {
				em.Params = append(em.Params, "...") // add vararg
			}
			em.Annotations = append(em.Annotations, d)
			*/
		} else {
			typeDoc = append(typeDoc, d)
		}
	}

	var isMember bool
	if tags["member"] != nil {
		isMember = true
	}
	parentMod := mod
	dps := &docPiece{
		Doc: typeDoc,
		FuncName: typeName,
		Interfacing: interfaces,
		IsInterface: inInterface,
		IsMember: isMember,
		IsType: true,
		ParentModule: parentMod,
		Fields: fields,
		Properties: properties,
	}

	typeTable[strings.ToLower(typeName)] = []string{parentMod, interfaces}

	return dps
}

func setupDoc(mod string, fun *doc.Func) *docPiece {
	docs := strings.TrimSpace(fun.Doc)
	tags, parts := getTagsAndDocs(docs)

	// i couldnt fit this into the condition below for some reason so here's a goto!
	if tags["member"] != nil {
		goto start
	}

	if (!strings.HasPrefix(fun.Name, prefix[mod]) && tags["interface"] == nil) || (strings.ToLower(fun.Name) == "loader" && tags["interface"] == nil) {
		return nil
	}

start:
	inInterface := tags["interface"] != nil
	var interfaces string
	funcsig := parts[0]
	doc := parts[1:]
	funcName := strings.TrimPrefix(fun.Name, prefix[mod])
	funcdoc := []string{}

	if inInterface {
		interfaces = tags["interface"][0].id
		funcName = interfaces + "." + strings.Split(funcsig, "(")[0]
	}
	em := emmyPiece{FuncName: funcName}

	fields := docPieceTag("field", tags)
	properties := docPieceTag("property", tags)

	for _, d := range doc {
		if strings.HasPrefix(d, "---") {
			emmyLine := strings.TrimSpace(strings.TrimPrefix(d, "---"))
			emmyLinePieces := strings.Split(emmyLine, " ")
			emmyType := emmyLinePieces[0]
			if emmyType == "@param" {
				em.Params = append(em.Params, emmyLinePieces[1])
			}
			if emmyType == "@vararg" {
				em.Params = append(em.Params, "...") // add vararg
			}
			em.Annotations = append(em.Annotations, d)
		} else {
			funcdoc = append(funcdoc, d)
		}
	}

	var isMember bool
	if tags["member"] != nil {
		isMember = true
	}
	var parentMod string
	if inInterface {
		parentMod = mod
	}
	dps := &docPiece{
		Doc: funcdoc,
		FuncSig: funcsig,
		FuncName: funcName,
		Interfacing: interfaces,
		GoFuncName: strings.ToLower(fun.Name),
		IsInterface: inInterface,
		IsMember: isMember,
		ParentModule: parentMod,
		Fields: fields,
		Properties: properties,
	}
	if strings.HasSuffix(dps.GoFuncName, strings.ToLower("loader")) {
		dps.Doc = parts
	}
	em.DocPiece = dps
			
	emmyDocs[mod] = append(emmyDocs[mod], em)
	return dps
}

func main() {
	fset := token.NewFileSet()
	os.Mkdir("docs", 0777)
	os.Mkdir("docs/api", 0777)
	os.Mkdir("emmyLuaDocs", 0777)

	dirs := []string{"./readline", "./"}
	filepath.Walk("golibs/", func (path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		dirs = append(dirs, "./" + path)
		return nil
	})

	pkgs := make(map[string]*ast.Package)
	for _, path := range dirs {
		d, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Println(err)
			return
		}
		for k, v := range d {
			pkgs[k] = v
		}
	}

	interfaceModules := make(map[string]*module)
	for l, f := range pkgs {
		p := doc.New(f, "./", doc.AllDecls)
		pieces := []docPiece{}
		typePieces := []docPiece{}
		mod := l
		if mod == "main" || mod == "readline" {
			mod = "hilbish"
		}
		var hasInterfaces bool
		for _, t := range p.Funcs {
			piece := setupDoc(mod, t)
			if piece == nil {
				continue
			}

			pieces = append(pieces, *piece)
			if piece.IsInterface {
				hasInterfaces = true
			}
		}
		for _, t := range p.Types {
			typePiece := setupDocType(mod, t)
			if typePiece != nil {
				typePieces = append(typePieces, *typePiece)
				if typePiece.IsInterface {
					hasInterfaces = true
				}
			}

			for _, m := range t.Methods {
				piece := setupDoc(mod, m)
				if piece == nil {
					continue
				}

				pieces = append(pieces, *piece)
				if piece.IsInterface {
					hasInterfaces = true
				}
			}
		}

		tags, descParts := getTagsAndDocs(strings.TrimSpace(p.Doc))
		shortDesc := descParts[0]
		desc := descParts[1:]
		filteredPieces := []docPiece{}
		filteredTypePieces := []docPiece{}
		for _, piece := range pieces {
			if !piece.IsInterface {
				filteredPieces = append(filteredPieces, piece)
				continue
			}

			modname := piece.ParentModule + "." + piece.Interfacing
			if interfaceModules[modname] == nil {
				interfaceModules[modname] = &module{
					ParentModule: piece.ParentModule,
				}
			}

			if strings.HasSuffix(piece.GoFuncName, strings.ToLower("loader")) {
				shortDesc := piece.Doc[0]
				desc := piece.Doc[1:]
				interfaceModules[modname].ShortDescription = shortDesc
				interfaceModules[modname].Description = strings.Join(desc, "\n")
				interfaceModules[modname].Fields = piece.Fields
				interfaceModules[modname].Properties = piece.Properties
				continue
			}

			interfaceModules[modname].Docs = append(interfaceModules[modname].Docs, piece)
		}

		for _, piece := range typePieces {
			if !piece.IsInterface {
				filteredTypePieces = append(filteredTypePieces, piece)
				continue
			}

			modname := piece.ParentModule + "." + piece.Interfacing
			if interfaceModules[modname] == nil {
				interfaceModules[modname] = &module{
					ParentModule: piece.ParentModule,
				}
			}

			interfaceModules[modname].Types = append(interfaceModules[modname].Types, piece)
		}

		docs[mod] = module{
			Types: filteredTypePieces,
			Docs: filteredPieces,
			ShortDescription: shortDesc,
			Description: strings.Join(desc, "\n"),
			HasInterfaces: hasInterfaces,
			Properties: docPieceTag("property", tags),
			Fields: docPieceTag("field", tags),
		}
	}

	for key, mod := range interfaceModules {
		docs[key] = *mod
	}

	var wg sync.WaitGroup
	wg.Add(len(docs) * 2)

	for mod, v := range docs {
		docPath := "docs/api/" + mod + ".md"
		if v.HasInterfaces {
			os.Mkdir("docs/api/" + mod, 0777)
			os.Remove(docPath) // remove old doc path if it exists
			docPath = "docs/api/" + mod + "/_index.md"
		}
		if v.ParentModule != "" {
			docPath = "docs/api/" + v.ParentModule + "/" + mod + ".md"
		}

		go func(modname, docPath string, modu module) {
			defer wg.Done()
			modOrIface := "Module"
			if modu.ParentModule != "" {
				modOrIface = "Interface"
			}

			f, _ := os.Create(docPath)
			f.WriteString(fmt.Sprintf(header, modOrIface, modname, modu.ShortDescription))
			typeTag, _ := regexp.Compile(`@\w+`)
			modDescription := typeTag.ReplaceAllStringFunc(strings.Replace(modu.Description, "<", `\<`, -1), func(typ string) string {
				typName := typ[1:]
				typLookup := typeTable[strings.ToLower(typName)]
				ifaces := typLookup[0] + "." + typLookup[1] + "/"
				if typLookup[1] == "" {
					ifaces = ""
				}
				linkedTyp := fmt.Sprintf("/Hilbish/docs/api/%s/%s#%s", typLookup[0], ifaces, strings.ToLower(typName))
				return fmt.Sprintf(`<a href="%s" style="text-decoration: none;">%s</a>`, linkedTyp, typName)
			})
			f.WriteString(fmt.Sprintf("## Introduction\n%s\n\n", modDescription))
			if len(modu.Fields) != 0 {
				f.WriteString("## Interface fields\n")
				for _, dps := range modu.Fields {
					f.WriteString(fmt.Sprintf("- `%s`: ", dps.FuncName))
					f.WriteString(strings.Join(dps.Doc, " "))
					f.WriteString("\n")
				}
				f.WriteString("\n")
			}
			if len(modu.Properties) != 0 {
				f.WriteString("## Object properties\n")
				for _, dps := range modu.Properties {
					f.WriteString(fmt.Sprintf("- `%s`: ", dps.FuncName))
					f.WriteString(strings.Join(dps.Doc, " "))
					f.WriteString("\n")
				}
				f.WriteString("\n")
			}

			if len(modu.Docs) != 0 {
				f.WriteString("## Functions\n")
				for _, dps := range modu.Docs {
					if dps.IsMember {
						continue
					}
					htmlSig := typeTag.ReplaceAllStringFunc(strings.Replace(dps.FuncSig, "<", `\<`, -1), func(typ string) string {
						typName := typ[1:]
						typLookup := typeTable[strings.ToLower(typName)]
						if len(typLookup) == 0 {
							typLookup = []string{"WHAT", "WHAT"}
						}
						ifaces := typLookup[0] + "." + typLookup[1] + "/"
						if typLookup[1] == "" {
							ifaces = ""
						}
						linkedTyp := fmt.Sprintf("/Hilbish/docs/api/%s/%s#%s", typLookup[0], ifaces, strings.ToLower(typName))
						return fmt.Sprintf(`<a href="%s" style="text-decoration: none;">%s</a>`, linkedTyp, typName)
					})
					f.WriteString(fmt.Sprintf("### %s\n", htmlSig))
					for _, doc := range dps.Doc {
						if !strings.HasPrefix(doc, "---") {
							f.WriteString(doc + "\n")
						}
					}
					f.WriteString("\n")
				}
			}

			if len(modu.Types) != 0 {
				f.WriteString("## Types\n")
				for _, dps := range modu.Types {
					f.WriteString(fmt.Sprintf("## %s\n", dps.FuncName))
					for _, doc := range dps.Doc {
						if !strings.HasPrefix(doc, "---") {
							f.WriteString(doc + "\n")
						}
					}
					if len(dps.Properties) != 0 {
						f.WriteString("### Properties\n")
						for _, dps := range dps.Properties {
							f.WriteString(fmt.Sprintf("- `%s`: ", dps.FuncName))
							f.WriteString(strings.Join(dps.Doc, " "))
							f.WriteString("\n")
						}
					}
					f.WriteString("\n")
					f.WriteString("### Methods\n")
					for _, dps := range modu.Docs {
						if !dps.IsMember {
							continue
						}
						htmlSig := typeTag.ReplaceAllStringFunc(strings.Replace(dps.FuncSig, "<", `\<`, -1), func(typ string) string {
							typName := regexp.MustCompile(`\w+`).FindString(typ[1:])
							typLookup := typeTable[strings.ToLower(typName)]
							fmt.Printf("%+q, \n", typLookup)
							linkedTyp := fmt.Sprintf("/Hilbish/docs/api/%s/%s/#%s", typLookup[0], typLookup[0] + "." + typLookup[1], strings.ToLower(typName))
							return fmt.Sprintf(`<a href="#%s" style="text-decoration: none;">%s</a>`, linkedTyp, typName)
						})
						f.WriteString(fmt.Sprintf("#### %s\n", htmlSig))
						for _, doc := range dps.Doc {
							if !strings.HasPrefix(doc, "---") {
								f.WriteString(doc + "\n")
							}
						}
						f.WriteString("\n")
					}
				}
			}
		}(mod, docPath, v)

		go func(md, modname string, modu module) {
			defer wg.Done()

			if modu.ParentModule != "" {
				return
			}

			ff, _ := os.Create("emmyLuaDocs/" + modname + ".lua")
			ff.WriteString("--- @meta\n\nlocal " + modname + " = {}\n\n")
			for _, em := range emmyDocs[modname] {
				if strings.HasSuffix(em.DocPiece.GoFuncName, strings.ToLower("loader")) {
					continue
				}

				dps := em.DocPiece
				funcdocs := dps.Doc
				ff.WriteString("--- " + strings.Join(funcdocs, "\n--- ") + "\n")
				if len(em.Annotations) != 0 {
					ff.WriteString(strings.Join(em.Annotations, "\n") + "\n")
				}
				accessor := "."
				if dps.IsMember {
					accessor = ":"
				}
				signature := strings.Split(dps.FuncSig, " ->")[0]
				var intrface string
				if dps.IsInterface {
					intrface = "." + dps.Interfacing
				}
				ff.WriteString("function " + modname + intrface + accessor + signature + " end\n\n")
			}
			ff.WriteString("return " + modname + "\n")
		}(mod, mod, v)
	}
	wg.Wait()
}
