package python_grammar

import (
	"github.com/stretchr/testify/require"

	"os"
	"path"
	"strings"
	"testing"
)

func TestImport(t *testing.T) {
	tests := []struct {
		Name                string
		ExpectedImports     []Import
		ExpectedFromImports []FromImport
	}{
		{
			Name:            "import foo",
			ExpectedImports: []Import{{Path: []string{"foo"}}},
		},
		{
			Name:            "import foo as alias",
			ExpectedImports: []Import{{Path: []string{"foo"}, Alias: "alias"}},
		},
		{
			Name:            "import foo.bar",
			ExpectedImports: []Import{{Path: []string{"foo", "bar"}}},
		},
		{
			Name:            "import foo.bar as alias",
			ExpectedImports: []Import{{Path: []string{"foo", "bar"}, Alias: "alias"}},
		},
		{
			Name:            "import foo   . bar",
			ExpectedImports: []Import{{Path: []string{"foo", "bar"}}},
		},
		{
			Name:            "  import foo",
			ExpectedImports: []Import{{Path: []string{"foo"}, Indented: true}},
		},
		{
			Name:            "   import foo.bar",
			ExpectedImports: []Import{{Path: []string{"foo", "bar"}, Indented: true}},
		},
		{
			Name:            "    import foo   . bar",
			ExpectedImports: []Import{{Path: []string{"foo", "bar"}, Indented: true}},
		},
		{
			Name:                "from foo import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo"}}},
		},
		{
			Name:                " from foo import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo"}, Indented: true}},
		},
		{
			Name:                "from foo import a, b",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (a)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (a, b)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (a,\n  b)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (\n    a,\n    b\n)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo.bar import (a)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo", "bar"}}},
		},
		{
			Name:                "from foo.bar import (a, b)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo", "bar"}}},
		},
		{
			Name:                "from foo.bar import (a,\n  b)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo", "bar"}}},
		},
		{
			Name:                "from foo.bar import (\n    a,\n    b\n)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b"}}, Path: []string{"foo", "bar"}}},
		},
		{
			Name:                "from foo import *",
			ExpectedFromImports: []FromImport{{All: true, Path: []string{"foo"}}},
		},
		{
			Name:                "from .foo import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo"}, Relative: []bool{true}}},
		},
		{
			Name:                "from.foo import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo"}, Relative: []bool{true}}},
		},
		{
			Name:                "from .foo.bar import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Path: []string{"foo", "bar"}, Relative: []bool{true}}},
		},
		{
			Name:                "from . import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Relative: []bool{true}}},
		},
		{
			Name:                "from ... import a",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}}, Relative: []bool{true, true, true}}},
		},
		{
			Name:                "from foo import a as a_2, b as b_2",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a", Alias: "a_2"}, {Name: "b", Alias: "b_2"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (a as a_2)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a", Alias: "a_2"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (a, b as b_2)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a"}, {Name: "b", Alias: "b_2"}}, Path: []string{"foo"}}},
		},
		{
			Name:                "from foo import (a as a_2,\n  b)",
			ExpectedFromImports: []FromImport{{Names: []ImportedName{{Name: "a", Alias: "a_2"}, {Name: "b"}}, Path: []string{"foo"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			a := require.New(t)

			var content []byte
			if strings.HasSuffix(tt.Name, ".py") {
				var err error
				content, err = os.ReadFile(path.Join(".import_test", tt.Name))
				a.NoError(err)
			} else {
				content = []byte(tt.Name)
			}
			parsed, err := parser.ParseBytes("", content)
			a.NoError(err)

			var imports []Import
			var fromImports []FromImport

			for _, stmt := range parsed.Statements {
				switch {
				case stmt.Import != nil:
					imports = append(imports, *stmt.Import)
				case stmt.FromImport != nil:
					fromImports = append(fromImports, *stmt.FromImport)
				}
			}

			a.Equal(tt.ExpectedImports, imports)
			a.Equal(tt.ExpectedFromImports, fromImports)
		})
	}
}