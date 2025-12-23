package web

import (
	"df/core"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"sync"
)

// var templatesFS embed.FS

type WebServer struct {
	app        *core.App
	port       int
	templates  map[string]*template.Template
	scanStatus struct {
		sync.Mutex
		Progress float64
		Scanning bool
		Results  []core.ResultList
		Error    error
	}
}

func NewWebServer(app *core.App, port int) *WebServer {
	ws := &WebServer{
		app:  app,
		port: port,
	}
	ws.loadTemplates()
	return ws
}

func (ws *WebServer) loadTemplates() {
	// For now, we'll use string templates to keep it simple and avoid embed issues during initial setup
	// but we'll plan for embed later.
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	// CSS for purple cloud theme
	css := `
		:root {
			--primary: #8a2be2;
			--primary-light: #f3e5f5;
			--secondary: #9c27b0;
			--text: #4a148c;
			--bg: #faf5ff;
			--card-bg: rgba(255, 255, 255, 0.8);
			--shadow: 0 8px 32px 0 rgba(138, 43, 226, 0.2);
		}
		body { 
			font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; 
			background: linear-gradient(135deg, #f3e5f5 0%, #e1bee7 100%); 
			background-attachment: fixed;
			color: var(--text); 
			margin: 0; 
			padding: 20px; 
			min-height: 100vh;
		}
		h1, h2, h3 { color: var(--primary); font-weight: 300; }
		.container { 
			max-width: 1200px; 
			margin: 0 auto; 
			background: var(--card-bg); 
			backdrop-filter: blur(10px);
			-webkit-backdrop-filter: blur(10px);
			padding: 30px; 
			border-radius: 20px; 
			box-shadow: var(--shadow);
			border: 1px solid rgba(255, 255, 255, 0.3);
		}
		.btn { 
			background: linear-gradient(45deg, #8a2be2, #9c27b0); 
			color: white; 
			border: none; 
			padding: 12px 24px; 
			border-radius: 12px; 
			cursor: pointer; 
			text-decoration: none; 
			display: inline-block; 
			margin: 5px; 
			transition: transform 0.2s, box-shadow 0.2s; 
			font-weight: 500;
			box-shadow: 0 4px 15px rgba(138, 43, 226, 0.3);
		}
		.btn:hover { 
			transform: translateY(-2px);
			box-shadow: 0 6px 20px rgba(138, 43, 226, 0.4);
		}
		.btn-danger { background: linear-gradient(45deg, #ff4081, #f50057); }
		.btn-secondary { background: linear-gradient(45deg, #7e57c2, #5e35b1); }
		table { width: 100%; border-collapse: separate; border-spacing: 0 10px; margin-top: 20px; }
		th { text-align: left; padding: 15px; color: var(--secondary); font-weight: 600; text-transform: uppercase; font-size: 0.8rem; letter-spacing: 1px; }
		td { padding: 15px; background: rgba(255, 255, 255, 0.5); border: none; }
		tr td:first-child { border-radius: 12px 0 0 12px; }
		tr td:last-child { border-radius: 0 12px 12px 0; }
		.progress-bar { width: 100%; background-color: rgba(138, 43, 226, 0.1); border-radius: 10px; margin: 20px 0; height: 10px; overflow: hidden; }
		.progress-inner { height: 100%; background: linear-gradient(90deg, #8a2be2, #ff4081); width: 0%; transition: width 0.5s ease-out; }
		.card { 
			background: rgba(255, 255, 255, 0.6); 
			padding: 20px; 
			border-radius: 15px; 
			margin-bottom: 20px; 
			border: 1px solid rgba(138, 43, 226, 0.1);
		}
		.dupe-group { 
			margin-bottom: 30px; 
			border-radius: 15px; 
			overflow: hidden; 
			box-shadow: 0 4px 15px rgba(0,0,0,0.05);
			background: white;
		}
		.dupe-header { 
			background: linear-gradient(90deg, #f3e5f5, #e1bee7); 
			color: var(--text); 
			padding: 15px 20px; 
			display: flex; 
			justify-content: space-between; 
			align-items: center;
			font-weight: 600;
		}
		.nav { 
			margin-bottom: 30px; 
			padding-bottom: 15px; 
			display: flex;
			gap: 10px;
		}
		.nav a { 
			color: var(--secondary); 
			text-decoration: none; 
			padding: 8px 16px;
			border-radius: 10px;
			transition: background 0.3s;
			font-weight: 500;
		}
		.nav a:hover { background: var(--primary-light); }
		input[type="text"] { 
			background: white; 
			border: 2px solid var(--primary-light); 
			color: var(--text); 
			padding: 12px; 
			border-radius: 12px; 
			width: 300px; 
			outline: none;
			transition: border-color 0.3s;
		}
		input[type="text"]:focus { border-color: var(--primary); }
		
		/* Toggle Switch */
		.switch {
			position: relative;
			display: inline-block;
			width: 50px;
			height: 24px;
		}
		.switch input { 
			opacity: 0;
			width: 0;
			height: 0;
		}
		.slider {
			position: absolute;
			cursor: pointer;
			top: 0;
			left: 0;
			right: 0;
			bottom: 0;
			background-color: #ccc;
			transition: .4s;
			border-radius: 24px;
		}
		.slider:before {
			position: absolute;
			content: "";
			height: 16px;
			width: 16px;
			left: 4px;
			bottom: 4px;
			background-color: white;
			transition: .4s;
			border-radius: 50%;
		}
		input:checked + .slider {
			background-color: var(--primary);
		}
		input:focus + .slider {
			box-shadow: 0 0 1px var(--primary);
		}
		input:checked + .slider:before {
			transform: translateX(26px);
		}

		/* Compact dupe groups */
		.dupe-group { 
			margin-bottom: 15px; 
			border-radius: 10px; 
			overflow: hidden; 
			box-shadow: 0 2px 8px rgba(0,0,0,0.05);
			background: white;
		}
		.dupe-header { 
			background: linear-gradient(90deg, #f3e5f5, #e1bee7); 
			color: var(--text); 
			padding: 8px 15px; 
			display: flex; 
			justify-content: space-between; 
			align-items: center;
			font-weight: 600;
			font-size: 0.9rem;
		}
		.dupe-group table td {
			padding: 8px 15px;
		}
		.btn-sm {
			padding: 4px 8px;
			font-size: 0.8rem;
			border-radius: 6px;
		}
		.align-right {
			text-align: right;
		}
		.selection-controls {
			margin-bottom: 15px;
			display: flex;
			gap: 10px;
		}
	`

	layout := `
<!DOCTYPE html>
<html>
<head>
	<title>DupeFiles Web</title>
	<style>` + css + `</style>
</head>
<body>
	<div class="container">
		<div class="nav">
			<a href="/">Dashboard</a>
			<a href="/index">Database</a>
			<a href="/results">Results</a>
			<a href="/config">Config</a>
		</div>
		{{block "content" .}}{{end}}
	</div>
	<script>
		function refreshProgress() {
			fetch('/api/status')
				.then(response => response.json())
				.then(data => {
					const pb = document.getElementById('progress-inner');
					if (pb) {
						pb.style.width = (data.Progress * 100) + '%';
						if (data.Scanning) {
							setTimeout(refreshProgress, 500);
						} else if (window.location.pathname === '/scan') {
							window.location.href = '/results';
						}
					}
				});
		}
		if (document.getElementById('progress-inner')) {
			refreshProgress();
		}
	</script>
</body>
</html>
`

	ws.templates = make(map[string]*template.Template)

	pages := map[string]string{
		"dashboard": `
{{define "content"}}
	<h1>Dashboard</h1>
	<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
		<div class="card">
			<h3>Quick Actions</h3>
			<a href="/scan" class="btn">Start New Scan</a>
			<a href="/index" class="btn btn-secondary">Manage Index</a>
		</div>
		<div class="card">
			<h3>Statistics</h3>
			<p>Total files in database: <b>{{.FilesCount}}</b></p>
			{{if .LastScanResults}}
				<p>Duplicates found in last scan: <b style="color: var(--secondary)">{{len .LastScanResults}}</b> groups</p>
			{{end}}
		</div>
	</div>
{{end}}
`,
		"index": `
{{define "content"}}
	<h1>Database Management</h1>
	<div class="card">
		<h3>Add Path</h3>
		<form action="/api/add-path" method="POST">
			<input type="text" name="path" placeholder="/path/to/scan" required>
			<button type="submit" class="btn">Add Path</button>
		</form>
	</div>
	<div class="card">
		<h3>Database Actions</h3>
		<a href="/api/purge" class="btn btn-secondary">Purge (Remove missing)</a>
		<a href="/api/update" class="btn btn-secondary">Update (Refresh hashes)</a>
		<a href="/api/clear" class="btn btn-danger" onclick="return confirm('Clear everything?')">Clear Index</a>
	</div>
	<h3>Indexed Files</h3>
	<table>
		<tr>
			<th>Path</th>
			<th>Size</th>
			<th>Hash</th>
			<th>Action</th>
		</tr>
		{{range .Files}}
		<tr>
			<td>{{.Path}}</td>
			<td>{{.Size}}</td>
			<td>{{if .Hash.Valid}}{{.Hash.String}}{{else}}<i>N/A</i>{{end}}</td>
			<td><a href="/api/remove-file?guid={{.Guid}}" class="btn btn-danger btn-sm">Remove</a></td>
		</tr>
		{{end}}
	</table>
{{end}}
`,
		"results": `
{{define "content"}}
	<h1>Scan Results</h1>
	<div class="nav">
		<a href="/api/move-all" class="btn btn-secondary">Move All to...</a>
		<a href="/api/trash-all" class="btn btn-danger" onclick="return confirm('Move all duplicates to trash?')">Trash All</a>
	</div>
	
	<div class="selection-controls">
		<button class="btn btn-sm" onclick="selectAll()">Select All</button>
		<button class="btn btn-sm" onclick="selectNone()">Select None</button>
		<button class="btn btn-sm" onclick="invertSelection()">Invert Selection</button>
		<button class="btn btn-sm" onclick="autoSelect()">Auto Select (Keep First)</button>
	</div>

	{{range $idx, $group := .Results}}
		<div class="dupe-group" data-group-id="{{$idx}}">
			<div class="dupe-header">
				<span>Group {{add $idx 1}} - Hash: {{.Hash}}</span>
				<span>Size: {{.Size}} bytes</span>
			</div>
			<table>
				{{range .Files}}
				<tr>
					<td width="30">
						<input type="checkbox" class="file-checkbox" data-guid="{{.Guid}}" data-group-id="{{$idx}}">
					</td>
					<td>{{.Path}}</td>
					<td class="align-right">
						<a href="/api/delete-file?guid={{.Guid}}" class="btn btn-danger btn-sm" onclick="return confirm('Delete this file?')">Delete</a>
					</td>
				</tr>
				{{end}}
			</table>
		</div>
	{{else}}
		<p>No results found. Start a scan first!</p>
	{{end}}
	
	<script>
		function selectAll() {
			document.querySelectorAll('.file-checkbox').forEach(cb => cb.checked = true);
		}
		function selectNone() {
			document.querySelectorAll('.file-checkbox').forEach(cb => cb.checked = false);
		}
		function invertSelection() {
			document.querySelectorAll('.file-checkbox').forEach(cb => cb.checked = !cb.checked);
		}
		function autoSelect() {
			// Logic from ui.go: keep first, select others in each group
			const groups = {};
			document.querySelectorAll('.file-checkbox').forEach(cb => {
				const gid = cb.getAttribute('data-group-id');
				if (!groups[gid]) {
					groups[gid] = [];
				}
				groups[gid].push(cb);
			});
			
			for (const gid in groups) {
				groups[gid].forEach((cb, index) => {
					cb.checked = (index > 0);
				});
			}
		}
	</script>
{{end}}
`,
		"scan": `
{{define "content"}}
	<h1>Scanning...</h1>
	<div class="progress-bar">
		<div id="progress-inner" class="progress-inner"></div>
	</div>
	<p>Please wait while we search for duplicate files.</p>
{{end}}
`,
		"config": `
{{define "content"}}
	<h1>Configuration</h1>
	<form action="/api/config" method="POST">
		<table>
			<tr>
				<th>Database Filename</th>
				<td>
					<div style="display: flex; gap: 5px;">
						<input type="text" name="db_filename" value="{{.Config.DBFilename}}" style="flex-grow: 1;">
						<button type="button" class="btn btn-sm" onclick="alert('File picker not available in browser. Please enter path manually.')">...</button>
					</div>
				</td>
			</tr>
			<tr>
				<th>Dry Run</th>
				<td>
					<label class="switch">
						<input type="checkbox" name="dry_run" {{if .Config.DryRun}}checked{{end}}>
						<span class="slider"></span>
					</label>
				</td>
			</tr>
			<tr>
				<th>Debug</th>
				<td>
					<label class="switch">
						<input type="checkbox" name="debug" {{if .Config.Debug}}checked{{end}}>
						<span class="slider"></span>
					</label>
				</td>
			</tr>
			<tr>
				<th>Min File Size (bytes)</th>
				<td>
					<input type="number" name="min_file_size" value="{{.Config.MinFileSize}}">
				</td>
			</tr>
		</table>
		<div style="margin-top: 20px;">
			<button type="submit" class="btn">Save Configuration</button>
		</div>
	</form>
{{end}}
`,
	}

	for name, content := range pages {
		tmpl := template.New(name).Funcs(funcMap)
		template.Must(tmpl.Parse(layout))
		template.Must(tmpl.Parse(content))
		ws.templates[name] = tmpl
	}
}

func (ws *WebServer) Start() error {
	http.HandleFunc("/", ws.handleDashboard)
	http.HandleFunc("/index", ws.handleIndex)
	http.HandleFunc("/results", ws.handleResults)
	http.HandleFunc("/config", ws.handleConfig)
	http.HandleFunc("/scan", ws.handleScan)

	// API
	http.HandleFunc("/api/status", ws.handleApiStatus)
	http.HandleFunc("/api/add-path", ws.handleApiAddPath)
	http.HandleFunc("/api/purge", ws.handleApiPurge)
	http.HandleFunc("/api/update", ws.handleApiUpdate)
	http.HandleFunc("/api/clear", ws.handleApiClear)
	http.HandleFunc("/api/remove-file", ws.handleApiRemoveFile)
	http.HandleFunc("/api/delete-file", ws.handleApiDeleteFile)
	http.HandleFunc("/api/trash-all", ws.handleApiTrashAll)
	http.HandleFunc("/api/config", ws.handleApiConfig)

	fmt.Printf("Starting webserver at http://localhost:%d\n", ws.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", ws.port), nil)
}

func (ws *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	files := ws.app.GetIndex().GetAllFiles()
	data := struct {
		FilesCount      int
		LastScanResults []core.ResultList
	}{
		FilesCount:      len(files),
		LastScanResults: ws.scanStatus.Results,
	}
	ws.render(w, "dashboard", data)
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	files := ws.app.GetIndex().GetAllFiles()
	// Sort files by path for better view
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	data := struct {
		Files []*core.FileItem
	}{
		Files: files,
	}
	ws.render(w, "index", data)
}

func (ws *WebServer) handleResults(w http.ResponseWriter, r *http.Request) {
	ws.scanStatus.Lock()
	results := ws.scanStatus.Results
	ws.scanStatus.Unlock()

	data := struct {
		Results []core.ResultList
	}{
		Results: results,
	}

	ws.render(w, "results", data)
}

func (ws *WebServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := ws.app.GetConfig()
	data := struct {
		Config *core.Config
	}{
		Config: cfg,
	}
	ws.render(w, "config", data)
}

func (ws *WebServer) handleScan(w http.ResponseWriter, r *http.Request) {
	ws.scanStatus.Lock()
	if ws.scanStatus.Scanning {
		ws.scanStatus.Unlock()
		ws.render(w, "scan", nil)
		return
	}
	ws.scanStatus.Scanning = true
	ws.scanStatus.Progress = 0
	ws.scanStatus.Unlock()

	go func() {
		scanner := core.NewScanner(ws.app.GetIndex())
		scanner.ProgressCallback = func(p float64) {
			ws.scanStatus.Lock()
			ws.scanStatus.Progress = p
			ws.scanStatus.Unlock()
		}
		results, err := scanner.ScanForDuplicates()
		ws.scanStatus.Lock()
		ws.scanStatus.Results = results
		ws.scanStatus.Error = err
		ws.scanStatus.Scanning = false
		ws.scanStatus.Unlock()
	}()

	ws.render(w, "scan", nil)
}

func (ws *WebServer) handleApiStatus(w http.ResponseWriter, r *http.Request) {
	ws.scanStatus.Lock()
	defer ws.scanStatus.Unlock()
	json.NewEncoder(w).Encode(ws.scanStatus)
}

func (ws *WebServer) handleApiAddPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.FormValue("path")
	if path != "" {
		_, err := ws.app.AddPathToIndex(path, true, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func (ws *WebServer) handleApiPurge(w http.ResponseWriter, r *http.Request) {
	ws.app.IndexPurge()
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func (ws *WebServer) handleApiUpdate(w http.ResponseWriter, r *http.Request) {
	ws.app.IndexUpdate()
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func (ws *WebServer) handleApiClear(w http.ResponseWriter, r *http.Request) {
	ws.app.IndexClear()
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func (ws *WebServer) handleApiRemoveFile(w http.ResponseWriter, r *http.Request) {
	guid := r.URL.Query().Get("guid")
	if guid != "" {
		ws.app.GetIndex().RemoveFile(guid)
	}
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func (ws *WebServer) handleApiDeleteFile(w http.ResponseWriter, r *http.Request) {
	guid := r.URL.Query().Get("guid")
	if guid != "" {
		file := ws.app.GetIndex().GetFileByGuid(guid)
		if file != nil {
			ws.app.DeleteFiles([]*core.FileItem{file})
		}
	}
	http.Redirect(w, r, "/results", http.StatusSeeOther)
}

func (ws *WebServer) handleApiTrashAll(w http.ResponseWriter, r *http.Request) {
	ws.app.MoveDuplicateFilesToTrash()
	http.Redirect(w, r, "/results", http.StatusSeeOther)
}

func (ws *WebServer) handleApiConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := ws.app.GetConfig()
	cfg.DBFilename = r.FormValue("db_filename")
	cfg.DryRun = r.FormValue("dry_run") == "on"
	cfg.Debug = r.FormValue("debug") == "on"
	if minSize, err := strconv.ParseInt(r.FormValue("min_file_size"), 10, 64); err == nil {
		cfg.MinFileSize = minSize
	}

	http.Redirect(w, r, "/config", http.StatusSeeOther)
}

func (ws *WebServer) render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := ws.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
