package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"mvdan.cc/sh/v3/syntax"
)

func Test() {
	var r io.Reader
	file, err := os.Open("/home/snkatvit/IdeaProjects/contain-sentry/Dockerfile")
	if err != nil {
		println("Error opening Dockerfile")
		println(err.Error())
		os.Exit(1)
	}
	r = file
	parse, err := parser.Parse(r)
	if err != nil {
		println("Error parsing Dockerfile")
		println(err.Error())
		os.Exit(1)
	}

	fmt.Printf("Parsed Dockerfile %v", parse)

	for _, node := range parse.AST.Children {

		v, err := instructions.ParseInstruction(node)
		if err != nil {
			println("Error parsing Dockerfile Instruction")
		}
		fmt.Printf("Parsed Dockerfile Instruction %v \n", v)

		if rc, ok := v.(*instructions.RunCommand); ok {
			for _, cmd := range rc.ShellDependantCmdLine.CmdLine {
				fmt.Printf("Running cmd %v \n", cmd)
				analyzeRunCommand(cmd)
			}
		}

		command, err := instructions.ParseCommand(node)
		if err != nil {
			println("Error parsing Dockerfile Command")
		}
		fmt.Printf("Parsed Dockerfile Command %v \n", command)

	}

}

func analyzeRunCommand(shellCode string) {
	// 1. Создаем парсер (можно настроить на Bash или POSIX)
	p := syntax.NewParser()

	// 2. Парсим строку в Shell-AST (f - это *syntax.File)
	f, err := p.Parse(strings.NewReader(shellCode), "dockerfile_run")
	if err != nil {
		// Если ошибка — значит в RUN невалидный shell (или специфичный синтаксис)
		return
	}

	// 3. Обходим дерево Shell-команд
	syntax.Walk(f, func(node syntax.Node) bool {
		switch x := node.(type) {

		case *syntax.CallExpr: // Это вызов команды (например, "apt-get install")
			cmdName := getCommandName(x)
			if cmdName == "sudo" {
				fmt.Println("CRITICAL: Использование sudo внутри контейнера запрещено!")
			}
			if cmdName == "curl" || cmdName == "wget" {
				// Тут можно проверить, нет ли пайпа в | sh
			}

		case *syntax.BinaryCmd: // Это пайпы (|) или логические операторы (&&, ||)
			if x.Op == syntax.Pipe {
				// Анализируем правую часть пайпа: x.Y
				// Если там sh/bash — это потенциальный RCE (curl ... | sh)
			}
		}
		return true
	})
}

// Вспомогательная функция для извлечения имени команды
func getCommandName(ce *syntax.CallExpr) string {
	if len(ce.Args) > 0 && len(ce.Args[0].Parts) > 0 {
		if lit, ok := ce.Args[0].Parts[0].(*syntax.Lit); ok {
			return lit.Value
		}
	}
	return ""
}
