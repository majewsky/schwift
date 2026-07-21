///usr/bin/env go run "$0" "$@"; exit $!

// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	failIfError(err)

	sections := strings.SplitN(string(input), "\n---\n", 2)
	if len(sections) != 2 {
		fail("missing separator between data and template")
	}
	dataStr, templateStr := sections[0], sections[1]

	data := make(map[string]any)
	failIfError(json.Unmarshal([]byte(dataStr), &data))

	tmpl, err := template.New("tmpl").Parse(templateStr)
	failIfError(err)

	failIfError(tmpl.Execute(os.Stdout, data))
}

func failIfError(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
