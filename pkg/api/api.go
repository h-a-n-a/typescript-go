package api

import (
	"C"
	"os"

	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	ts "github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs/vfstest"
)

func TranspileModule(source string) string {
	// Parse compiler options
	compilerOptions := &core.CompilerOptions{
		ModuleKind:      core.ModuleKindESNext,
		Target:          core.ScriptTargetESNext,
		IsolatedModules: core.TSTrue,
		ESModuleInterop: core.TSTrue,
		OutDir:          "/dist",
		OutFile:         "/dist/main.js",
	}

	// Set up the compiler environment
	fs := bundled.WrapFS(vfstest.FromMap(
		map[string]any{
			"/main.ts": source,
		},
		false,
	))

	defaultLibraryPath := bundled.LibPath()
	currentDirectory, _ := os.Getwd()

	// Create a compiler host
	host := ts.NewCompilerHost(compilerOptions, currentDirectory, fs, defaultLibraryPath)

	program := ts.NewProgram(ts.ProgramOptions{
		ConfigFileName: "",
		RootFiles:      []string{"/main.ts"},
		Options:        compilerOptions,
		SingleThreaded: true,
		Host:           host,
	})

	result := program.Emit(&ts.EmitOptions{
		TargetSourceFile: program.GetSourceFileByPath(tspath.Path("/main.ts")),
	})

	if len(result.Diagnostics) > 0 {
		printDiagnostics(result.Diagnostics, host, compilerOptions)
	}

	content, _ := fs.ReadFile("/dist/main.js")

	return content
}

func printDiagnostic(d *ast.Diagnostic, level int, comparePathOptions tspath.ComparePathsOptions) {
	file := d.File()
	if file != nil {
		p := tspath.ConvertToRelativePath(file.FileName(), comparePathOptions)
		line, character := scanner.GetLineAndCharacterOfPosition(file, d.Loc().Pos())
		fmt.Printf("%v%v(%v,%v): error TS%v: %v\n", strings.Repeat(" ", level*2), p, line+1, character+1, d.Code(), d.Message())
	} else {
		fmt.Printf("%verror TS%v: %v\n", strings.Repeat(" ", level*2), d.Code(), d.Message())
	}
	printMessageChain(d.MessageChain(), level+1)
	for _, r := range d.RelatedInformation() {
		printDiagnostic(r, level+1, comparePathOptions)
	}
}

func printMessageChain(messageChain []*ast.Diagnostic, level int) {
	for _, c := range messageChain {
		fmt.Printf("%v%v\n", strings.Repeat(" ", level*2), c.Message())
		printMessageChain(c.MessageChain(), level+1)
	}
}

func getFormatOpts(host ts.CompilerHost) *diagnosticwriter.FormattingOptions {
	return &diagnosticwriter.FormattingOptions{
		NewLine: host.NewLine(),
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          host.GetCurrentDirectory(),
			UseCaseSensitiveFileNames: host.FS().UseCaseSensitiveFileNames(),
		},
	}
}

func printDiagnostics(diagnostics []*ast.Diagnostic, host ts.CompilerHost, compilerOptions *core.CompilerOptions) {
	formatOpts := getFormatOpts(host)
	if compilerOptions.Pretty.IsTrueOrUnknown() {
		diagnosticwriter.FormatDiagnosticsWithColorAndContext(os.Stdout, diagnostics, formatOpts)
		fmt.Fprintln(os.Stdout)
		diagnosticwriter.WriteErrorSummaryText(os.Stdout, diagnostics, formatOpts)
	} else {
		for _, diagnostic := range diagnostics {
			printDiagnostic(diagnostic, 0, formatOpts.ComparePathsOptions)
		}
	}
}
