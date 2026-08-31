package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestResolveStaticDir_PrefersViewsStatic(t *testing.T) {
	tmp := t.TempDir()
	require.Equal(t, filepath.Join(tmp, "static"), resolveStaticDir(tmp))

	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "views", "static"), 0o755))
	require.Equal(t, filepath.Join(tmp, "views", "static"), resolveStaticDir(tmp))
}

func TestResolveStaticDir_IgnoresViewsStaticFile(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "views"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "views", "static"), []byte("not-a-dir"), 0o644))
	require.Equal(t, filepath.Join(tmp, "static"), resolveStaticDir(tmp))
}

func TestFileServer_ServesViewsStatic(t *testing.T) {
	tmp := t.TempDir()
	viewsDir := filepath.Join(tmp, "views")
	viewsStatic := filepath.Join(viewsDir, "static")
	require.NoError(t, os.MkdirAll(viewsStatic, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(viewsDir, "index.html"), []byte("<html>ok</html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(viewsStatic, "app.js"), []byte("from-views"), 0o644))

	// 旧的 ./static 不应抢占 views/static
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "static", "app.js"), []byte("from-root"), 0o644))

	router := chi.NewRouter()
	(&dynamicMux{}).mountStaticFiles(router, tmp)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "from-views", rec.Body.String())

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<html>ok</html>")
}

func TestFileServer_FallsBackToRootStatic(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "views"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "static", "app.js"), []byte("from-root"), 0o644))

	router := chi.NewRouter()
	(&dynamicMux{}).mountStaticFiles(router, tmp)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "from-root", rec.Body.String())
}

