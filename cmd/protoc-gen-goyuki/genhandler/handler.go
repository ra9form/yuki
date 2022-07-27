package genhandler

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/ra9form/yuki/cmd/protoc-gen-goyuki/internal"
	"github.com/ra9form/yuki/cmd/protoc-gen-goyuki/third-party/grpc-gateway/internals/descriptor"
	"golang.org/x/tools/imports"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	plugin "google.golang.org/protobuf/types/pluginpb"
)

type Generator struct {
	options options
	reg     *descriptor.Registry
	imports []descriptor.GoPackage // common imports
}

// New returns a new generator which generates handler wrappers.
func New(reg *descriptor.Registry, opts ...Option) *Generator {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	g := &Generator{
		options: o,
		reg:     reg,
	}
	return g
}

func (g *Generator) newGoPackage(pkgPath string, aalias ...string) descriptor.GoPackage {
	gopkg := descriptor.GoPackage{
		Path: pkgPath,
		Name: path.Base(pkgPath),
	}
	alias := gopkg.Name
	if len(aalias) > 0 {
		alias = aalias[0]
		gopkg.Alias = alias
	}

	reference := alias
	if reference == "" {
		reference = gopkg.Name
	}

	for i := 0; ; i++ {
		if err := g.reg.ReserveGoPackageAlias(alias, gopkg.Path); err == nil {
			break
		}
		alias = fmt.Sprintf("%s_%d", gopkg.Name, i)
		gopkg.Alias = alias
	}

	if pkg == nil {
		pkg = make(map[string]string)
	}
	pkg[reference] = alias

	return gopkg
}

func (g *Generator) generateDesc(file *descriptor.File) (*plugin.CodeGeneratorResponse_File, error) {
	descCode, err := g.getDescTemplate(g.options.SwaggerDef[file.GetName()], file)

	if err != nil {
		return nil, err
	}
	name := filepath.Base(file.GetName())
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	prefix := g.pathPrefixFor(file)
	output := fmt.Sprintf(filepath.Join(prefix, "%s.pb.goyuki.go"), base)
	output = filepath.Clean(output)

	// removing redundant imports from desc files (tested while generating integration/no_bindings/pb/strings.pb.goyuki.go)
	descCode, err = prettifyCode(output, descCode)
	if err != nil {
		glog.Errorf("%v: %s", err, annotateString(descCode))
		return nil, err
	}

	glog.V(1).Infof("Will emit %s", output)

	return &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(output),
		Content: proto.String(descCode),
	}, nil
}

func (g *Generator) generateSwaggerFile(file *descriptor.File) *plugin.CodeGeneratorResponse_File {
	if g.options.SwaggerPath == "" || len(g.options.SwaggerDef) == 0 {
		return nil
	}

	swaggerContent := g.options.SwaggerDef[file.GetName()]
	if swaggerContent == nil {
		return nil
	}

	name := filepath.Base(file.GetName())
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	output := fmt.Sprintf(filepath.Join(g.options.SwaggerPath, "%s.json"), base)
	output = filepath.Clean(output)

	glog.V(1).Infof("Will emit %s", output)

	return &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(output),
		Content: proto.String(string(swaggerContent)),
	}
}

func (g *Generator) generateImpl(file *descriptor.File) (files []*plugin.CodeGeneratorResponse_File, err error) {
	guessModule()
	prefix := g.pathPrefixFor(file)
	implPathRelative := g.implPathRelativeFor(file)

	var pkg *ast.Package
	if !g.options.ServiceSubDir {
		pkg, err = astPkg(descriptor.GoPackage{
			Name: file.GoPkg.Name,
			Path: filepath.Join(prefix, g.options.ImplPath),
		})
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			pkg, err = astPkg(descriptor.GoPackage{
				Name: file.GoPkg.Name,
				Path: filepath.Join(prefix, implPathRelative),
			})
			if err != nil {
				return nil, err
			}
		}
	}
	for _, svc := range file.Services {
		if g.options.ServiceSubDir {
			pkg, err = astPkg(descriptor.GoPackage{
				Name: file.GoPkg.Name,
				Path: filepath.Join(prefix, g.options.ImplPath, internal.KebabCase(svc.GetName())),
			})
			if err != nil {
				return nil, err
			}
			if pkg == nil {
				pkg, err = astPkg(descriptor.GoPackage{
					Name: file.GoPkg.Name,
					Path: filepath.Join(prefix, implPathRelative, internal.KebabCase(svc.GetName())),
				})
				if err != nil {
					return nil, err
				}
			}
		}
		if code, err := g.generateImplService(file, svc, pkg); err == nil {
			files = append(files, code...)
		} else {
			return nil, err
		}
	}
	return files, nil
}

func (g *Generator) generateImplService(file *descriptor.File, svc *descriptor.Service, astPkg *ast.Package) ([]*plugin.CodeGeneratorResponse_File, error) {
	var files []*plugin.CodeGeneratorResponse_File

	if exists := astTypeExists(implTypeName(svc), astPkg); !exists || g.options.Force {
		prefix := g.pathPrefixFor(file)
		var output string
		if g.options.ServiceSubDir {
			output = fmt.Sprintf(filepath.Join(prefix, g.options.ImplPath, internal.KebabCase(svc.GetName()), "%s.go"), implFileName(svc, nil))
		} else {
			output = fmt.Sprintf(filepath.Join(prefix, g.options.ImplPath, "%s.go"), implFileName(svc, nil))
		}
		implCode, err := g.getServiceImpl(file, svc)

		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(implCode))
		if err != nil {
			glog.Errorf("%v: %s", err, annotateString(implCode))
			return nil, err
		}

		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(string(formatted)),
		})

		glog.V(1).Infof("Will emit %s", output)
	} else {
		glog.V(0).Infof("Implementation of service `%s` will not be emitted: type `%s` already exists in package `%s`", svc.GetName(), implTypeName(svc), file.GoPkg.Name)
	}

	for _, method := range svc.Methods {
		if code, err := g.generateImplServiceMethod(file, svc, method, astPkg); err == nil {
			files = append(files, code...)
		} else {
			return nil, err
		}
	}

	return files, nil
}

func (g *Generator) generateImplServiceMethod(file *descriptor.File, svc *descriptor.Service, method *descriptor.Method, astPkg *ast.Package) ([]*plugin.CodeGeneratorResponse_File, error) {
	methodGoName := goTypeName(method.GetName())
	if exists := astMethodExists(implTypeName(svc), methodGoName, astPkg); !exists || g.options.Force {
		prefix := g.pathPrefixFor(file)
		var output string
		if g.options.ServiceSubDir {
			output = fmt.Sprintf(filepath.Join(prefix, g.options.ImplPath, internal.KebabCase(svc.GetName()), "%s.go"), implFileName(svc, method))
		} else {
			output = fmt.Sprintf(filepath.Join(prefix, g.options.ImplPath, "%s.go"), implFileName(svc, method))
		}
		output = filepath.Clean(output)
		implCode, err := g.getMethodImpl(svc, method)
		if err != nil {
			return nil, err
		}
		implCode, err = prettifyCode(output, implCode)
		if err != nil {
			glog.Errorf("%v: %s", err, annotateString(implCode))
			return nil, err
		}

		glog.V(1).Infof("Will emit %s", output)

		result := []*plugin.CodeGeneratorResponse_File{{
			Name:    proto.String(output),
			Content: proto.String(implCode),
		}}

		if !g.options.WithTests {
			return result, nil
		}

		testCode, err := g.getTestImpl(svc, method)
		if err != nil {
			return nil, err
		}

		// removing redundant imports from test files (tested while generating integration/google_empty/strings/empty_response_test.go)
		testCode, err = prettifyCode(output, testCode)
		if err != nil {
			glog.Errorf("%v: %s", err, annotateString(testCode))
			return nil, err
		}

		result = append(result, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(strings.TrimSuffix(output, ".go") + "_test.go"),
			Content: proto.String(testCode),
		})

		return result, nil
	}
	glog.V(0).Infof("Implementation of method `%s` for service `%s` will not be emitted: method already exists in package: `%s`", methodGoName, svc.GetName(), file.GoPkg.Name)

	return nil, nil
}

func (g *Generator) Generate(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var files []*plugin.CodeGeneratorResponse_File
	for _, file := range targets {
		glog.V(1).Infof("Processing %s", file.GetName())

		if len(file.Services) == 0 {
			glog.V(0).Infof("%s: %v", file.GetName(), errNoTargetService)
			continue
		}

		if code, err := g.generateDesc(file); err == nil {
			files = append(files, code)
		} else {
			return nil, err
		}

		if swaggerFile := g.generateSwaggerFile(file); swaggerFile != nil {
			files = append(files, swaggerFile)
		}

		if g.options.Impl {
			if code, err := g.generateImpl(file); err == nil {
				files = append(files, code...)
			} else {
				return nil, err
			}
		}
	}

	return files, nil
}

func (g *Generator) getDescTemplate(swagger []byte, f *descriptor.File) (string, error) {
	pkgSeen := make(map[string]bool)
	var allImports []descriptor.GoPackage
	for _, pkg := range g.imports {
		pkgSeen[pkg.Path] = true
		allImports = append(allImports, pkg)
	}

	pkgs := []string{
		"fmt",
		"io/ioutil",
		"strings",
		"bytes",
		"net/http",
		"net/url",
		"encoding/base64",
		"context",

		"github.com/ra9form/yuki/transport/httpruntime",
		"github.com/ra9form/yuki/transport/httptransport",
		"github.com/ra9form/yuki/transport/swagger",
		"github.com/grpc-ecosystem/grpc-gateway/v2/runtime",
		"github.com/grpc-ecosystem/grpc-gateway/v2/utilities",
		"google.golang.org/grpc",
		"github.com/go-chi/chi",
		"github.com/pkg/errors",
		"github.com/ra9form/yuki/transport",
	}

	if swagger != nil {
		pkgs = append(pkgs, "github.com/go-openapi/spec")
	}

	for _, pkg := range pkgs {
		pkgSeen[pkg] = true
		allImports = append(allImports, g.newGoPackage(pkg))
	}

	httpmw := g.newGoPackage("github.com/ra9form/yuki/transport/httpruntime/httpmw")
	httpcli := g.newGoPackage("github.com/ra9form/yuki/transport/httpclient")
	for _, svc := range f.Services {
		for _, m := range svc.Methods {
			checkedAppend := func(pkg descriptor.GoPackage) {
				// Add request type package to imports if needed
				if m.Options == nil || !proto.HasExtension(m.Options, annotations.E_Http) ||
					pkg == f.GoPkg || pkgSeen[pkg.Path] {
					return
				}
				pkgSeen[pkg.Path] = true

				// always generate alias for external packages, when types used in req/resp object
				if pkg.Alias == "" {
					pkg.Alias = pkg.Name
					pkgSeen[pkg.Path] = false
				}

				allImports = append(allImports, pkg)
			}

			checkedAppend(m.RequestType.File.GoPkg)
			checkedAppend(m.ResponseType.File.GoPkg)
		}

		if hasBindings(svc) && !pkgSeen[httpcli.Path] {
			allImports = append(allImports, httpcli)
			pkgSeen[httpcli.Path] = true
		}

		if g.options.ApplyDefaultMiddlewares && hasBindings(svc) && !pkgSeen[httpmw.Path] {
			allImports = append(allImports, httpmw)
			pkgSeen[httpmw.Path] = true
		}
	}

	p := param{
		File:             f,
		Imports:          allImports,
		ApplyMiddlewares: g.options.ApplyDefaultMiddlewares,
		Registry:         g.reg,
	}

	if swagger != nil {
		p.SwaggerBuffer = swagger
	}

	return applyDescTemplate(p)
}

func (g *Generator) getServiceImpl(f *descriptor.File, s *descriptor.Service) (string, error) {
	// restore orig GoPkg
	savedPkg := f.GoPkg
	defer func() {
		f.GoPkg = savedPkg
	}()
	return applyImplTemplate(g.getImplParam(f, s, nil, []string{"github.com/ra9form/yuki/transport"}))
}

func (g *Generator) getMethodImpl(s *descriptor.Service, m *descriptor.Method) (string, error) {
	// restore orig GoPkg
	savedPkg := s.File.GoPkg
	defer func() {
		s.File.GoPkg = savedPkg
	}()

	return applyImplTemplate(g.getImplParam(s.File, s, m, []string{"context", "github.com/pkg/errors"}))
}

func (g *Generator) getTestImpl(s *descriptor.Service, m *descriptor.Method) (string, error) {
	// restore orig GoPkg
	savedPkg := s.File.GoPkg
	defer func() {
		s.File.GoPkg = savedPkg
	}()

	return applyTestTemplate(g.getImplParam(s.File, s, m, []string{"context", "testing", "github.com/stretchr/testify/require"}))
}

func (g *Generator) getImplParam(f *descriptor.File, s *descriptor.Service, m *descriptor.Method, deps []string) implParam {
	pkgSeen := make(map[string]bool)
	var imports []descriptor.GoPackage
	for _, pkg := range g.imports {
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
	}
	for _, pkg := range deps {
		pkgSeen[pkg] = true
		imports = append(imports, g.newGoPackage(pkg))
	}
	p := implParam{
		ImplGoPkgPath: f.GoPkg.Path,
		Service:       s,
		Method:        m,
		File:          f,
	}
	fileGoPkg := f.GoPkg
	if g.options.ImplPath != "" {
		descImport := getDescImportPath(f)
		p.ImplGoPkgPath = filepath.Join(descImport, g.options.ImplPath)

		var (
			cleanedPkgPathRequest  string
			cleanedPkgPathResponse string
		)
		if m != nil {
			// fixing relative paths in proto file (like option go_package = "./pb;strings";).
			cleanedPkgPathRequest = strings.TrimPrefix(m.RequestType.File.GoPkg.Path, "./")
			cleanedPkgPathResponse = strings.TrimPrefix(m.ResponseType.File.GoPkg.Path, "./")
		}

		// Generate desc imports only if need
		if m != nil &&
			strings.Index(cleanedPkgPathRequest, "/") >= 0 && !strings.HasSuffix(descImport, cleanedPkgPathRequest) &&
			strings.Index(cleanedPkgPathResponse, "/") >= 0 && !strings.HasSuffix(descImport, cleanedPkgPathResponse) {
		} else {
			// set relative f.GoPkg for proper determining package for types from desc import
			// f.GoPkg uses in function .Method.RequestType.GoType
			f.GoPkg = g.newGoPackage(descImport, "desc")
			f.GoPkg.Name = fileGoPkg.Name
			pkgSeen[f.GoPkg.Path] = true
			imports = append(imports, f.GoPkg)
		}
	}
	if m != nil {
		checkedAppend := func(pkg descriptor.GoPackage) {
			if pkg.Path == fileGoPkg.Path || pkgSeen[pkg.Path] {
				return
			}
			pkgSeen[pkg.Path] = true

			// always generate alias for external packages, when types used in req/resp object
			if pkg.Alias == "" {
				pkg.Alias = pkg.Name
				pkgSeen[pkg.Path] = false
			}

			imports = append(imports, pkg)
		}
		checkedAppend(m.RequestType.File.GoPkg)
		checkedAppend(m.ResponseType.File.GoPkg)
	}

	p.Imports = imports
	return p
}

// protobuf-v2 has an option to store generated pb relative to protofile path, which is widely used.
// 	More on implementation details - google.golang.org/protobuf@v1.27.1/compiler/protogen/protogen.go:432.
func (g *Generator) pathPrefixFor(file *descriptor.File) string {
	if g.options.PathsParamType == PathsParamTypeSourceRelative {
		return strings.TrimSuffix(file.GeneratedFilenamePrefix, filepath.Base(file.GeneratedFilenamePrefix))
	}
	return file.GoPkg.Path
}

func (g *Generator) implPathRelativeFor(file *descriptor.File) string {
	if g.options.PathsParamType == PathsParamTypeSourceRelative {
		return filepath.Join(file.GoPkg.Path, g.options.ImplPath)
	}
	return g.options.ImplPath
}

func annotateString(str string) string {
	strs := strings.Split(str, "\n")
	for pos := range strs {
		strs[pos] = fmt.Sprintf("%v: %v", pos, strs[pos])
	}
	return strings.Join(strs, "\n")
}

// prettifyCode is applying gofmt and goimports to the file contents given.
func prettifyCode(filePath string, code string) (string, error) {
	formatted, err := format.Source([]byte(code))
	if err != nil {
		return "", err
	}

	formatted, err = imports.Process(filePath, formatted, nil)
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

func getDescImportPath(file *descriptor.File) string {
	// wd is current working directory
	wd, err := filepath.Abs(".")
	if err != nil {
		glog.V(-1).Info(err)
	}
	// xwd = wd but after symlink evaluation
	xwd, direrr := filepath.EvalSymlinks(wd)

	// if we know module
	if module != "" {
		return getImportPath(file.GoPkg, wd, "")
	}

	for _, gp := range strings.Split(build.Default.GOPATH, ":") {
		gp = filepath.Clean(gp)
		// xgp = gp but after symlink evaluation
		xgp, gperr := filepath.EvalSymlinks(gp)
		if strings.HasPrefix(wd, gp) {
			return getImportPath(file.GoPkg, wd, gp)
		}
		if direrr == nil && strings.HasPrefix(xwd, gp) {
			return getImportPath(file.GoPkg, xwd, gp)
		}
		if gperr == nil && strings.HasPrefix(wd, xgp) {
			return getImportPath(file.GoPkg, wd, xgp)
		}
		if gperr == nil && direrr == nil && strings.HasPrefix(xwd, xgp) {
			return getImportPath(file.GoPkg, xwd, xgp)
		}
	}
	return ""
}

// getImportPath returns full go import path for specified gopkg
// wd - current working directory
// gopath - current gopath (can be empty if you are not in gopath)
func getImportPath(goPackage descriptor.GoPackage, wd, gopath string) string {
	var wdImportPath, gopkg string
	if goPackage.Path != "." {
		gopkg = goPackage.Path
	}
	if module != "" && moduleDir != "" {
		wdImportPath = filepath.Join(module, strings.TrimPrefix(wd, moduleDir))
	} else {
		wdImportPath = strings.TrimPrefix(wd, filepath.Join(gopath, "src")+string(filepath.Separator))
	}
	if strings.HasPrefix(gopkg, wdImportPath) {
		return gopkg
	} else if gopkg != "" {
		return filepath.Join(wdImportPath, gopkg)
	} else {
		return wdImportPath
	}
}

func hasBindings(service *descriptor.Service) bool {
	for _, m := range service.Methods {
		if len(m.Bindings) > 0 {
			return true
		}
	}
	return false
}

func astPkg(pkg descriptor.GoPackage) (*ast.Package, error) {
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkg.Path, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") &&
			!strings.HasSuffix(name, "_test.go") && strings.HasSuffix(name, ".go")
	}, parser.DeclarationErrors)
	if filterError(err) != nil {
		return nil, err
	}
	return astPkgs[pkg.Name], nil
}

func filterError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*os.PathError); ok {
		return nil
	}
	return err
}

func astTypeExists(typeName string, pkg *ast.Package) bool {
	if pkg == nil {
		return false
	}
	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			if gd, ok := d.(*ast.GenDecl); ok {
				for _, s := range gd.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && ts.Name != nil && ts.Name.Name == typeName {
						return true
					}
				}
			}
		}
	}
	return false
}

func astMethodExists(typeName, methodName string, pkg *ast.Package) bool {
	if pkg == nil {
		return false
	}
	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Name != nil && fd.Name.Name == methodName && fd.Recv != nil && len(fd.Recv.List) > 0 {
				if se, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok {
					if i, ok := se.X.(*ast.Ident); ok && i.Name == typeName {
						return true
					}
				}
			}
		}
	}
	return false
}

var module string
var moduleDir string
var moduleRegExp = regexp.MustCompile("^module (.*?)(?: //.*)?$")

func guessModule() {
	// dir is current working directory
	dir, err := filepath.Abs(".")
	if err != nil {
		glog.V(-1).Info(err)
	}

	// try to find go.mod
	mod := ""
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			mod = filepath.Join(root, "go.mod")
			break
		}
		if root == "" {
			break
		}
		d := filepath.Dir(root)
		if d == root {
			break
		}
		root = d
	}

	// if go.mod found
	if mod != "" {
		glog.V(1).Infof("Found mod file: %s", mod)
		fd, err := os.Open(mod)
		if err != nil {
			glog.V(-1).Info(err)
		}
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			line := scanner.Bytes()
			if matches := moduleRegExp.FindSubmatch(line); len(matches) > 1 {
				module = string(matches[1])
				moduleDir = root
				glog.V(1).Infof("Current module: %s", module)
				glog.V(1).Infof("Project directory: %s", moduleDir)
			}
		}
	}
}
