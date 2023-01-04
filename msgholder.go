package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type TranslationString struct {
	Position string // filename:line
	Singular string
	Plural   string
	Context  string
	Domain   string
}

type MsgHolder struct {
	strings map[string][]TranslationString
}

func (h *MsgHolder) Add(s TranslationString) {
	domain := s.Domain
	if domain == "" {
		domain = "default"
	}
	h.strings[domain] = append(h.strings[domain], s)
}

func getRows(str string) string {
	rows := strings.Split(str, "\n")
	var ret string

	if len(rows) == 0 {
		return str
	}

	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	for k, row := range rows {
		row = replacer.Replace(row)
		if k == len(rows)-1 {
			ret += fmt.Sprintf("\"%s\"\n", row)
			continue
		}
		ret += fmt.Sprintf("\"%s\\n\"\n", row)
	}
	return ret
}

func (h *MsgHolder) WriteOutput(outputFolder string, languages []string) error {
	for _, lang := range languages {
		langFolder := filepath.Join(outputFolder, lang)
		st, err := os.Stat(langFolder)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(langFolder, 0755)
				if err != nil {
					return fmt.Errorf("could not create folder '%s': %w", langFolder, err)
				}
			} else {
				return fmt.Errorf("stat returned error for folder '%s': %w", langFolder, err)
			}
		} else if !st.IsDir() {
			return fmt.Errorf("path %s already exists, but is not a directory", langFolder)
		}

		for domain := range h.strings {
			domainPath := filepath.Join(langFolder, fmt.Sprintf("%s.po", domain))

			poFileExists := false
			st, err = os.Stat(domainPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("stat returned error for domain path '%s': %w", domainPath, err)
				}
			} else {
				if st.Mode().IsRegular() {
					poFileExists = true
				} else {
					return fmt.Errorf("cannot update path %s - is not a file", domainPath)
				}
			}

			// First, create a temporary path to store the pot-file
			fd, err := ioutil.TempFile("", "domain.*.pot")
			if err != nil {
				return fmt.Errorf("could not create temporary domain file: %w", err)
			}

			fmt.Fprint(fd, header)
			h.WriteDomain(fd, domain)

			tempfileName := fd.Name()

			// Then, run msguniq on the file
			cmdArgs := []string{"msguniq", "--to-code=utf-8", "-o", tempfileName, tempfileName}

			stderr := bytes.Buffer{}
			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("could not run command '%s': %w\n%s", strings.Join(cmdArgs, " "), err, stderr.String())
			}

			if poFileExists {
				// Then, run msgmerge if PO-file already exists
				cmdArgs = []string{"msgmerge", "-N", "-q", "--previous", "-o", domainPath, domainPath, tempfileName}
				stderr := bytes.Buffer{}
				cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
				cmd.Stderr = &stderr
				err = cmd.Run()
				if err != nil {
					return fmt.Errorf("could not run command '%s': %w\n%s", strings.Join(cmdArgs, " "), err, stderr.String())
				}
			} else {
				newPo, err := os.Create(domainPath)
				if err != nil {
					return fmt.Errorf("could not create file '%s': %w", domainPath, err)
				}

				_, err = fd.Seek(0, io.SeekStart)
				if err != nil {
					return err
				}

				_, err = io.Copy(newPo, fd)
				if err != nil {
					return err
				}
				newPo.Close()
			}
			fd.Close()
		}

	}
	return nil
}

func (h *MsgHolder) WriteDomain(f *os.File, domain string) {
	dStrs := h.strings[domain]
	sort.Slice(dStrs, func(i, j int) bool {
		if dStrs[i].Context == dStrs[j].Context {
			return dStrs[i].Position < dStrs[j].Position
		}
		return dStrs[i].Context < dStrs[j].Context
	})
	for _, s := range dStrs {

		// Ignore empty strings - the empty string is reserved for translation information
		if s.Singular == "" {
			continue
		}

		fmt.Fprintf(f, "\n#: %s\n", s.Position)

		if s.Context != "" {
			fmt.Fprintf(f, "msgctxt \"%s\"\n", s.Context)
		}

		fmt.Fprintf(f, "msgid %s", getRows(s.Singular))

		if s.Plural != "" {
			fmt.Fprintf(f, "msgid_plural %s", getRows(s.Plural))
			fmt.Fprintf(f, "msgstr[0] \"\"\n")
			fmt.Fprintf(f, "msgstr[1] \"\"\n")
		} else {
			fmt.Fprintf(f, "msgstr \"\"\n")
		}
	}
}
