// Source code: https://doc.xuwenliang.com/docs/go/1814

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
)

var (
	flagRootDir    = flag.String("dir", "", "webdav root dir")
	flagHttpAddr   = flag.String("http", ":80", "http or https address")
	flagHttpsMode  = flag.Bool("https-mode", false, "use https mode")
	flagCertFile   = flag.String("https-cert-file", "cert.pem", "https cert file")
	flagKeyFile    = flag.String("https-key-file", "key.pem", "https key file")
	flagUserName   = flag.String("user", "", "user name")
	flagPassword   = flag.String("password", "", "user password")
	flagReadonly   = flag.Bool("read-only", false, "read only")
	flagShowHidden = flag.Bool("show-hidden", false, "show hidden files")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of WebDAV Server\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nReport bugs to <chaishushan@gmail.com>.\n")
	}
}

type SkipBrokenLink struct {
	webdav.Dir
}

func (d SkipBrokenLink) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fileinfo, err := d.Dir.Stat(ctx, name)
	if err != nil && os.IsNotExist(err) {
		return nil, filepath.SkipDir
	}
	return fileinfo, err
}

func main() {
	flag.Parse()
	fs := &webdav.Handler{
		FileSystem: SkipBrokenLink{webdav.Dir(*flagRootDir)},
		LockSystem: webdav.NewMemLS(),
	}
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if *flagUserName != "" && *flagPassword != "" {
			username, password, ok := req.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if username != *flagUserName || password != *flagPassword {
				http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
				return
			}
		}
		if req.Method == "GET" && handleDirList(fs.FileSystem, w, req) {
			return
		}
		if *flagReadonly {
			switch req.Method {
			case "PUT", "DELETE", "PROPPATCH", "MKCOL", "COPY", "MOVE":
				http.Error(w, "WebDAV: Read Only!!!", http.StatusForbidden)
				return
			}
		}
		fs.ServeHTTP(w, req)
	})
	if *flagHttpsMode {
		http.ListenAndServeTLS(*flagHttpAddr, *flagCertFile, *flagKeyFile, nil)
	} else {
		http.ListenAndServe(*flagHttpAddr, nil)
	}
}

func handleDirList(fs webdav.FileSystem, w http.ResponseWriter, req *http.Request) bool {
	ctx := context.Background()
	f, err := fs.OpenFile(ctx, req.URL.Path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {
		return false
	}
	if !strings.HasSuffix(req.URL.Path, "/") {
		http.Redirect(w, req, req.URL.Path+"/", 302)
		return true
	}
	dirs, err := f.Readdir(-1)
	if err != nil {
		log.Print(w, "Error reading directory", http.StatusInternalServerError)
		return false
	}

	// Sort 排序，目录 -> 文件
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i].IsDir() && !dirs[j].IsDir() {
			return true
		}
		if !dirs[i].IsDir() && dirs[j].IsDir() {
			return false
		}
		return dirs[i].Name() < dirs[j].Name()
	})

	folderName := filepath.Base(req.URL.Path)
	currentDir := req.URL.Path

	// 分割路径，创建目录路径导航
	parts := strings.Split(currentDir, "/")
	var navLinks []string
	for i := 1; i < len(parts); i++ {
		navPath := "/" + strings.Join(parts[1:i+1], "/")
		navLinks = append(navLinks, fmt.Sprintf(`<a href="%s">%s</a>`, navPath, parts[i]))
	}

	// 创建目录路径导航字符串
	nav := fmt.Sprintf(`
	<header>
	<div class="wrapper"><div class="breadcrumbs">Folder Path</div>
			<h1>
			<a href="/">/</a>%s
			</h1>
		</div>
	</header>
	`, strings.Join(navLinks, " / "))

	// Caddy2 file_server 样式
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>%s</title>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
			* { padding: 0; margin: 0; box-sizing: border-box; }

			body {
				font-family: Inter, system-ui, sans-serif;
				font-size: 16px;
				text-rendering: optimizespeed;
				background-color: #f3f6f7;
				min-height: 100vh;
			}

			img,
			svg {
				vertical-align: middle;
				z-index: 1;
			}

			img {
				max-width: 100%%;
				max-height: 100%%;
				border-radius: 5px;
			}

			td img {
				max-width: 1.5em;
				max-height: 2em;
				object-fit: cover;
			}

			body,
			a,
			svg,
			.layout.current,
			.layout.current svg,
			.go-up {
				color: #333;
				text-decoration: none;
			}

			.wrapper {
				max-width: 1200px;
				margin-left: auto;
				margin-right: auto;
			}

			header,
			.meta {
				padding-left: 5%%;
				padding-right: 5%%;
			}

			td a {
				color: #006ed3;
				text-decoration: none;
			}

			td a:hover {
				color: #0095e4;
			}

			th:first-child,
			td:first-child {
				width: 5%%;
			}

			th:last-child,
			td:last-child {
				width: 5%%;
			}

			.size,
			.timestamp {
				font-size: 14px;
			}

			header {
				padding-top: 15px;
				padding-bottom: 15px;
				box-shadow: 0px 0px 20px 0px rgb(0 0 0 / 10%%);
			}

			.breadcrumbs {
				text-transform: uppercase;
				font-size: 10px;
				letter-spacing: 1px;
				color: #939393;
				margin-bottom: 5px;
				padding-left: 3px;
			}

			h1 {
				font-size: 20px;
				font-family: Poppins, system-ui, sans-serif;
				font-weight: normal;
				white-space: nowrap;
				overflow-x: hidden;
				text-overflow: ellipsis;
				color: #c5c5c5;
			}

			h1 a,
			th a {
				color: #000;
			}

			h1 a {
				padding: 0 3px;
				margin: 0 1px;
			}

			h1 a:hover {
				background: #ffffc4;
			}

			h1 a:first-child {
				margin: 0;
			}

			header,
			main {
				background-color: white;
			}

			main {
				margin: 3em auto 0;
				border-radius: 5px;
				box-shadow: 0 2px 5px 1px rgb(0 0 0 / 5%%);
			}

			.meta {
				display: flex;
				gap: 1em;
				font-size: 14px;
				padding-top: 1em;
				padding-bottom: 0.2em;
			}

			#summary {
				display: flex;
				gap: 1em;
				align-items: center;
				margin-right: auto;
			}

			.layout,
			.layout svg {
				color: #9a9a9a;
			}

			table {
				width: 100%%;
				border-collapse: collapse;
			}

			tbody tr,
			tbody tr a,
			.entry a {
				transition: all .15s;
			}

			tbody tr:hover,
			.grid .entry a:hover {
				background-color: #f4f9fd;
			}

			th,
			td {
				text-align: left;
			}

			th {
				position: sticky;
				top: 0;
				background: white;
				white-space: nowrap;
				z-index: 2;
				text-transform: uppercase;
				font-size: 14px;
				letter-spacing: 1px;
				padding: .75em 0;
			}

			td {
				white-space: nowrap;
			}

			td:nth-child(2) {
				width: 75%%;
			}

			td:nth-child(2) a {
				padding: 1em 0;
				display: block;
			}

			td:nth-child(3),
			th:nth-child(3) {
				padding: 0 20px 0 20px;
				min-width: 150px;
			}

			td .go-up {
				text-transform: uppercase;
				font-size: 12px;
				font-weight: bold;
			}

			.name,
			.go-up {
				word-break: break-all;
				overflow-wrap: break-word;
				white-space: pre-wrap;
			}

			span.name {
				margin-left: 0.3em;
			}

			.listing .icon-tabler {
				color: #454545;
			}

			.listing .icon-tabler-folder-filled {
				color: #ffb900 !important;
			}

			footer {
				padding: 40px 20px;
				font-size: 12px;
				text-align: center;
			}

			@media (max-width: 600px) {
				.hideable {
					display: none;
				}

				td:nth-child(2) {
					width: auto;
				}

				th:nth-child(3),
				td:nth-child(3) {
					padding-right: 5%%;
					text-align: right;
				}

				h1 {
					color: #000;
				}

				h1 a {
					margin: 0;
				}
			}

			@media (prefers-color-scheme: dark) {
				html {
					background: black;
				}

				body {
					background: linear-gradient(180deg, rgb(34 50 66) 0%%, rgb(26 31 38) 100%%);
					background-attachment: fixed;
				}

				body,
				a,
				svg,
				.layout.current,
				.layout.current svg,
				.go-up {
					color: #ccc;
				}

				h1 a,
				th a {
					color: white;
				}

				h1 {
					color: white;
				}

				h1 a:hover {
					background: hsl(213deg 100%% 73%% / 20%%);
				}

				header,
				main,
				.grid .entry {
					background-color: #101720;
				}

				tbody tr:hover,
				.grid .entry a:hover {
					background-color: #162030;
					color: #fff;
				}

				th {
					background-color: #18212c;
				}

				td a,
				.listing .icon-tabler {
					color: #abc8e3;
				}

				td a:hover,
				td a:hover .icon-tabler {
					color: white;
				}

				#Wordmark path,
				#R path {
					fill: #ccc !important;
				}
				#R circle {
					stroke: #ccc !important;
				}
			}
			</style>
		</head>
		<body>
			%s
			<div class="wrapper">
			<main>
				<div class="meta">
				</div>
				<div class="listing">
				<table aria-describedby="summary">
				<thead>
					<tr>
						<th></th>
						<th>Name</th>
						<th class="size">Size</th>
						<th class="timestamp hideable">Modified</th>
						<th class="hideable"></th>
					</tr>
				</thead>
				<tbody>`, folderName, nav)
	if req.URL.Path != "/" {
		fmt.Fprintf(w, "<tr><td></td><td><a href=\"../\"><svg xmlns=\"http://www.w3.org/2000/svg\" class=\"icon icon-tabler icon-tabler-corner-left-up\" width=\"24\" height=\"24\" viewBox=\"0 0 24 24\" stroke-width=\"2\" stroke=\"currentColor\" fill=\"none\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path stroke=\"none\" d=\"M0 0h24v24H0z\" fill=\"none\"></path><path d=\"M18 18h-6a3 3 0 0 1 -3 -3v-10l-4 4m8 0l-4 -4\"></path></svg><span class=\"go-up\">Up</span></a></td></tr>\n")
	}
	for _, d := range dirs {
		if !*flagShowHidden && strings.HasPrefix(d.Name(), ".") {
			continue
		}
		link := d.Name()
		if d.IsDir() {
			link += "/"
		}
		name := link
		if d.IsDir() {
			fmt.Fprintf(w, "<tr class=\"file\"><td></td><td><a href=\"%s\"><svg xmlns=\"http://www.w3.org/2000/svg\" class=\"icon icon-tabler icon-tabler-folder-filled\" width=\"24\" height=\"24\" viewBox=\"0 0 24 24\" stroke-width=\"2\" stroke=\"currentColor\" fill=\"none\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path stroke=\"none\" d=\"M0 0h24v24H0z\" fill=\"none\"></path><path d=\"M9 3a1 1 0 0 1 .608 .206l.1 .087l2.706 2.707h6.586a3 3 0 0 1 2.995 2.824l.005 .176v8a3 3 0 0 1 -2.824 2.995l-.176 .005h-14a3 3 0 0 1 -2.995 -2.824l-.005 -.176v-11a3 3 0 0 1 2.824 -2.995l.176 -.005h4z\" stroke-width=\"0\" fill=\"#ffb900\"></path></svg><span class=\"name\">%s</span></a></td>", link, name)
			fmt.Fprintf(w, "<td>—</td>")
		} else {
			fmt.Fprintf(w, "<tr class=\"file\"><td></td><td><a href=\"%s\"><svg xmlns=\"http://www.w3.org/2000/svg\" class=\"icon icon-tabler icon-tabler-file\" width=\"24\" height=\"24\" viewBox=\"0 0 24 24\" stroke-width=\"2\" stroke=\"currentColor\" fill=\"none\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><path stroke=\"none\" d=\"M0 0h24v24H0z\" fill=\"none\"></path><path d=\"M14 3v4a1 1 0 0 0 1 1h4\"></path><path d=\"M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z\"></path></svg><span class=\"name\">%s</span></a></td>", link, name)
			fmt.Fprintf(w, "<td class=\"size\">%s</td>", formatSize(d.Size()))
		}
		fmt.Fprintf(w, "<td class=\"timestamp hideable\">%s</td>", d.ModTime().Format("2006/01/02 15:04:05"))
		fmt.Fprintln(w, "<td class=\"hideable\"></td></tr>")
	}
	fmt.Fprintf(w, `
				</tbody>
				</table>
				</div>
			</main>
			</div>
		</body>
		<footer></footer>
		</html>`)
	return true
}

func formatSize(bytes int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
