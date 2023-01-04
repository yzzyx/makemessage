package main

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

var (
	templatePaths      = pflag.StringSliceP("template-paths", "t", []string{}, "paths to template directories to parse")
	templateExtensions = pflag.StringSliceP("template-extensions", "e", []string{".html"}, "extensions of template files")
	packagePaths       = pflag.StringSliceP("package-paths", "p", []string{}, "paths to go packages to parse (use '.' to parse the current directory)")
	recurse            = pflag.BoolP("recursive", "r", false, "recurse into sub-packages")
	outputPath         = pflag.StringP("output", "o", "locales", "directory to place message files in")
	languages          = pflag.StringSliceP("languages", "l", []string{}, "languages to process")

	header = `# SOME DESCRIPTIVE TITLE.
# Copyright (C) YEAR THE PACKAGE'S COPYRIGHT HOLDER
# This file is distributed under the same license as the PACKAGE package.
# FIRST AUTHOR <EMAIL@ADDRESS>, YEAR.
#
#, fuzzy
msgid ""
msgstr ""
"Project-Id-Version: PACKAGE VERSION\n"
"Report-Msgid-Bugs-To: \n"
"POT-Creation-Date: 2019-11-29 14:11+0000\n"
"PO-Revision-Date: YEAR-MO-DA HO:MI+ZONE\n"
"Last-Translator: FULL NAME <EMAIL@ADDRESS>\n"
"Language-Team: LANGUAGE <LL@li.org>\n"
"Language: \n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Content-Transfer-Encoding: 8bit\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"
`
)

func main() {
	var err error

	pflag.Parse()

	msgHolder := &MsgHolder{
		strings: map[string][]TranslationString{},
	}

	if len(*languages) == 0 {
		fmt.Println("At least one language must be specified")
		return
	}

	if len(*packagePaths) == 0 && len(*templatePaths) == 0 {
		fmt.Println("At least one package path or template path must be specified")
		return

	}

	for _, p := range *packagePaths {
		folderList := []string{p}

		basePath, err := filepath.Abs(p)
		if err != nil {
			fmt.Printf("Cannot get absolute path of %s: %v\n", p, err)
			return
		}

		if *recurse {
			// Include all underlying packages as well
			err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
				if !info.Mode().IsDir() {
					return nil
				}

				if strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}

				if _, err = build.ImportDir(path, 0); err != nil {
					if _, noGo := err.(*build.NoGoError); !noGo {
						log.Print(err)
					}
					return nil
				}
				folderList = append(folderList, path)
				return nil
			})
			if err != nil {
				fmt.Println("Error recursing into folders:", err)
				return
			}
		}
		err = parseGo(basePath, folderList, msgHolder)
		if err != nil {
			fmt.Println("Error parsing packages:", err)
			return
		}
	}

	allowedExtensions := map[string]bool{}
	for _, ext := range *templateExtensions {
		allowedExtensions[ext] = true
	}

	for _, p := range *templatePaths {
		err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if !info.Mode().IsRegular() {
				return nil
			}

			if !allowedExtensions[filepath.Ext(path)] {
				return nil
			}

			return parseTemplate(path, msgHolder)
		})

		if err != nil {
			fmt.Println("cannot process files:", err)
			return
		}
	}

	err = msgHolder.WriteOutput(*outputPath, *languages)
	if err != nil {
		fmt.Println("Cannot create messages:", err)
		return
	}
}
