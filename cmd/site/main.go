// Package main implements the SIREN website server.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ht "html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aohorodnyk/mimeheader"
	"github.com/bcmk/siren-site/v3/sitelib"
	"github.com/bcmk/siren/lib/cmdlib"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tdewolff/minify/v2"
	hmin "github.com/tdewolff/minify/v2/html"

	_ "image/png"
)

type server struct {
	cfg          *sitelib.Config
	enabledPacks []sitelib.PackV2
	db           *pgxpool.Pool
	packs        []sitelib.PackV2

	enIndexTemplate     *ht.Template
	ruIndexTemplate     *ht.Template
	enStreamerTemplate              *ht.Template
	ruStreamerTemplate              *ht.Template
	enStreamerNotificationsTemplate *ht.Template
	ruStreamerNotificationsTemplate *ht.Template
	enStreamerChannelTemplate       *ht.Template
	ruStreamerChannelTemplate       *ht.Template
	enChicTemplate      *ht.Template
	ruChicTemplate      *ht.Template
	enPackTemplate      *ht.Template
	ruPackTemplate      *ht.Template
	enCodeTemplate      *ht.Template
	ruCodeTemplate      *ht.Template
	bioHeaderRemover    string
	partialFaviconsHTML string
	cssContent string
}

type likeForPack struct {
	Pack string `yaml:"pack"`
	Like bool   `yaml:"like"`
}

var funcMap = ht.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	"map": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, errors.New("invalid map call")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, errors.New("map keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	},
	"enhance": func(m1, m2 map[string]interface{}) map[string]interface{} {
		dict := map[string]interface{}{}
		for k, v := range m1 {
			dict[k] = v
		}
		for k, v := range m2 {
			dict[k] = v
		}
		return dict
	},
	"raw_html": func(h string) ht.HTML {
		return ht.HTML(h)
	},
	"mul_div": func(x, y, z int) int {
		return x * y / z
	},
	"add": func(x, y int) int {
		return x + y
	},
	"mul": func(a, b int) int {
		return a * b
	},
	"sub": func(x, y int) int {
		return x - y
	},
	"div": func(x, y int) int {
		return x / y
	},
	"versioned": func(pack *sitelib.PackV2, name string) string {
		icon := pack.Icons[name]
		if icon.Version == 0 {
			return name
		}
		return name + ".v" + strconv.Itoa(icon.Version)
	},
	"make_slice": func(xs ...any) []any { return xs },
	"atoi": func(s string) int {
		if s == "" {
			return 0
		}
		n, err := strconv.Atoi(s)
		checkErr(err)
		return n
	},
}

var packParams = []string{
	"siren",
	"fanclub",
	"instagram",
	"twitter",
	"onlyfans",
	"amazon",
	"lovense",
	"gift",
	"pornhub",
	"dmca",
	"allmylinks",
	"onemylink",
	"linktree",
	"fancentro",
	"frisk",
	"fansly",
	"throne",
	"avn",
	"mail",
	"snapchat",
	"telegram",
	"whatsapp",
	"youtube",
	"tiktok",
	"reddit",
	"twitch",
	"discord",
	"fanberry",
	"placement",
	"size",
}

var chaturbateModelRegex = regexp.MustCompile(`^(?:https?://)?(?:www\.|ar\.|de\.|el\.|en\.|es\.|fr\.|hi\.|it\.|ja\.|ko\.|nl\.|pt\.|ru\.|tr\.|zh\.|m\.)?chaturbate\.com(?:/p|/b)?/([A-Za-z0-9\-_@]+)/?(?:\?.*)?$|^([A-Za-z0-9\-_@]+)$`)

func linf(format string, v ...interface{}) { log.Printf("[INFO] "+format, v...) }
func ldbg(format string, v ...interface{}) { log.Printf("[DBG] "+format, v...) }

var checkErr = cmdlib.CheckErr

func notFoundError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, "404 not found")
}

func parseHTMLTemplate(filenames ...string) *ht.Template {
	var relative []string
	for _, f := range filenames {
		relative = append(relative, "pages/"+f)
	}
	t, err := ht.New(filepath.Base(filenames[0])).Funcs(funcMap).ParseFiles(relative...)
	checkErr(err)
	return t
}

func langs(url url.URL, baseDomain string, ls map[string]string) map[string]ht.URL {
	res := map[string]ht.URL{}
	port := url.Port()
	if port != "" {
		port = ":" + port
	}
	for l, pref := range ls {
		url.Host = pref + baseDomain + port
		res[l] = ht.URL(url.String())
	}
	return res
}

func getLangBaseURL(url url.URL, baseDomain string, baseURL string) string {
	if !strings.HasSuffix(url.Hostname(), baseDomain) {
		return baseURL
	}
	return "https://" + url.Host
}

func (s *server) tparams(r *http.Request, more map[string]interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	urlCopy := *r.URL
	res["full_path"] = urlCopy.String()
	urlCopy.Host = r.Host
	res["base_url"] = ht.URL(s.cfg.BaseURL)
	res["lang_base_url"] = ht.URL(getLangBaseURL(urlCopy, s.cfg.BaseDomain, s.cfg.BaseURL))
	res["hostname"] = urlCopy.Hostname()
	res["base_domain"] = s.cfg.BaseDomain
	res["ru_domain"] = "ru." + s.cfg.BaseDomain
	res["lang"] = langs(urlCopy, s.cfg.BaseDomain, map[string]string{"en": "", "ru": "ru."})
	res["version"] = cmdlib.Version
	for k, v := range more {
		res[k] = v
	}
	ah := mimeheader.ParseAcceptHeader(r.Header.Get("Accept"))
	imgExts := map[string]string{}
	imgExts["svg"] = "svgz"
	if ah.Match("image/webp") {
		imgExts["png"] = "webp"
	} else {
		imgExts["png"] = "svgz"
	}
	res["img_exts"] = imgExts
	res["chic_bucket_url"] = s.cfg.BaseBucketURL
	res["assets_bucket_url"] = s.cfg.AssetsBucketURL
	res["partial_favicons_html"] = s.partialFaviconsHTML
	res["css"] = ht.CSS(s.cssContent)

	return res
}

func (s *server) enIndexHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.enIndexTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) ruIndexHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.ruIndexTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) enStreamerHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.enStreamerTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) ruStreamerHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.ruStreamerTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) enStreamerNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.enStreamerNotificationsTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) ruStreamerNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.ruStreamerNotificationsTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) enStreamerChannelHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.enStreamerChannelTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) ruStreamerChannelHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.ruStreamerChannelTemplate.Execute(w, s.tparams(r, nil)))
}

func (s *server) enChicHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.enChicTemplate.Execute(w, s.tparams(r, map[string]interface{}{"packs": s.enabledPacks, "likes": s.likes()})))
}

func (s *server) ruChicHandler(w http.ResponseWriter, r *http.Request) {
	checkErr(s.ruChicTemplate.Execute(w, s.tparams(r, map[string]interface{}{"packs": s.enabledPacks, "likes": s.likes()})))
}

func (s *server) packHandler(w http.ResponseWriter, r *http.Request, t *ht.Template) {
	pack := s.findPack(mux.Vars(r)["pack"])
	if pack == nil {
		notFoundError(w)
		return
	}
	sirenError := false
	paramDict := getParamDict(packParams, r)
	siren := paramDict["siren"]
	if siren != "" && checkSirenParam(siren) == "" {
		sirenError = true
	}
	checkErr(t.Execute(w, s.tparams(r, map[string]interface{}{"pack": pack, "params": paramDict, "likes": s.likesForPack(pack.Name), "siren_error": sirenError})))
}

func (s *server) enPackHandler(w http.ResponseWriter, r *http.Request) {
	s.packHandler(w, r, s.enPackTemplate)
}

func (s *server) ruPackHandler(w http.ResponseWriter, r *http.Request) {
	s.packHandler(w, r, s.ruPackTemplate)
}

func checkSirenParam(siren string) string {
	m := chaturbateModelRegex.FindStringSubmatch(siren)
	if len(m) == 3 {
		siren = m[1]
		if siren == "" {
			siren = m[2]
		}
	}
	if siren == "in" || siren == "p" || siren == "b" || siren == "affiliates" || siren == "external_link" {
		return ""
	}
	return siren
}

func (s *server) codeHandler(w http.ResponseWriter, r *http.Request, t *ht.Template) {
	pack := s.findPack(mux.Vars(r)["pack"])
	if pack == nil {
		notFoundError(w)
		return
	}
	paramDict := getParamDict(packParams, r)
	siren := checkSirenParam(paramDict["siren"])
	var sizeErr error
	if paramDict["size"] != "" {
		_, sizeErr = strconv.Atoi(paramDict["size"])
	}
	if siren == "" || sizeErr != nil {
		target := "/chic/p/" + pack.Name
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
		return
	}
	paramDict["siren"] = siren
	code, err := s.chaturbateCode(pack, paramDict)
	if err != nil {
		notFoundError(w)
		return
	}
	checkErr(t.Execute(w, s.tparams(r, map[string]interface{}{"pack": pack, "params": paramDict, "code": code})))
}

func (s *server) enCodeHandler(w http.ResponseWriter, r *http.Request) {
	s.codeHandler(w, r, s.enCodeTemplate)
}

func (s *server) ruCodeHandler(w http.ResponseWriter, r *http.Request) {
	s.codeHandler(w, r, s.ruCodeTemplate)
}

func (s *server) testHandler(w http.ResponseWriter, r *http.Request) {
	pack := s.findPack(mux.Vars(r)["pack"])
	if pack == nil {
		notFoundError(w)
		return
	}
	paramDict := getParamDict(packParams, r)
	code, err := s.chaturbateCode(pack, paramDict)
	if err != nil {
		notFoundError(w)
		return
	}
	_, _ = w.Write([]byte(code))
}

func (s *server) likeHandler(w http.ResponseWriter, r *http.Request) {
	pack := s.findPack(mux.Vars(r)["pack"])
	if pack == nil {
		notFoundError(w)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1000))
	if err != nil {
		notFoundError(w)
		return
	}
	cmdlib.CloseBody(r.Body)
	var like likeForPack
	if err := json.Unmarshal(body, &like); err != nil {
		notFoundError(w)
		return
	}
	if like.Pack != pack.Name {
		notFoundError(w)
		return
	}
	ip := r.Header.Get("X-Forwarded-For")
	s.mustExec(`
		insert into likes (address, pack, "like", timestamp) values ($1, $2, $3, $4)
		on conflict(address, pack) do update set "like"=excluded."like", timestamp=excluded.timestamp`,
		ip,
		like.Pack,
		like.Like,
		int32(time.Now().Unix()),
	)
}

func (s *server) likes() map[string]int {
	query := s.mustQuery(`select pack, sum(case when "like" then 1 else -1 end) from likes group by pack`)
	defer query.Close()
	results := map[string]int{}
	for query.Next() {
		var pack string
		var count int
		checkErr(query.Scan(&pack, &count))
		results[pack] = count
	}
	return results
}

func (s *server) findPack(name string) *sitelib.PackV2 {
	for _, pack := range s.packs {
		if pack.Name == name {
			return &pack
		}
	}
	return nil
}

type iconSize struct {
	Width  float64
	Height float64
}

func (s *server) chaturbateCode(pack *sitelib.PackV2, params map[string]string) (string, error) {
	t := parseHTMLTemplate("common/icons-code-generator.gohtml")
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	unscaledSize := 5.4
	if params["size"] != "" {
		pxSize, err := strconv.Atoi(params["size"])
		if err != nil {
			return "", err
		}
		chaturbateREMToPXCoeff := 10.
		unscaledSize = float64(pxSize) / chaturbateREMToPXCoeff
	}
	width := unscaledSize * float64(*pack.ChaturbateIconsScale) / float64(100)
	hgap := 25
	if pack.HGap != nil {
		hgap = *pack.HGap
	}
	iconSizes := map[string]iconSize{}
	for k, v := range pack.Icons {
		iconSizes[k] = iconSize{
			Width:  width,
			Height: width * v.Height / v.Width,
		}
	}
	checkErr(t.Execute(w, map[string]interface{}{
		"pack":               pack,
		"params":             params,
		"hgap":               int(width*10) * (hgap + 100 - *pack.ChaturbateIconsScale) / 100,
		"base_url":           s.cfg.BaseURL,
		"icon_sizes":         iconSizes,
		"bio_header_remover": s.bioHeaderRemover,
	}))
	checkErr(w.Flush())
	m := minify.New()
	m.Add("text/html", &hmin.Minifier{KeepQuotes: true, KeepComments: true})
	str, err := m.String("text/html", b.String())
	if err != nil {
		panic(err)
	}
	return str, nil
}

func cacheControlHandler(h http.Handler, mins int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", mins*60))
		h.ServeHTTP(w, r)
	})
}

func (s *server) measure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		h.ServeHTTP(w, r)
		elapsed := time.Since(now).Milliseconds()
		if s.cfg.Debug {
			ldbg("performance for %s: %dms", r.URL.Path, elapsed)
		}
	})
}

func (s *server) likesForPack(pack string) int {
	return s.mustInt(`select coalesce(sum(case when "like" then 1 else -1 end), 0) from likes where pack = $1`, pack)
}

func (s *server) iconsCount() int {
	count := 0
	for _, i := range s.packs {
		count += len(i.Icons)
	}
	return count
}

func (s *server) fillRawFiles() {
	content, err := os.ReadFile("pages/common/bio-header-remover.html")
	checkErr(err)
	s.bioHeaderRemover = strings.TrimSuffix(string(content), "\n")
	re := regexp.MustCompile(`\n\s*`)
	s.bioHeaderRemover = re.ReplaceAllString(s.bioHeaderRemover, " ")
	re = regexp.MustCompile(`\(\s+`)
	s.bioHeaderRemover = re.ReplaceAllString(s.bioHeaderRemover, "(")
	re = regexp.MustCompile(`\s+\)`)
	s.bioHeaderRemover = re.ReplaceAllString(s.bioHeaderRemover, ")")

	content, err = os.ReadFile("partial/favicons.partial.html")
	checkErr(err)
	s.partialFaviconsHTML = string(content)

	content, err = os.ReadFile("static/style.css")
	checkErr(err)
	s.cssContent = string(content)
}

func (s *server) fillTemplates() {
	common := []string{"common/head.gohtml", "common/header.gohtml", "common/footer.gohtml", "common/header-icon.gohtml"}
	s.enIndexTemplate = parseHTMLTemplate(append([]string{"en/index.gohtml", "en/trans.gohtml"}, common...)...)
	s.ruIndexTemplate = parseHTMLTemplate(append([]string{"ru/index.gohtml", "ru/trans.gohtml"}, common...)...)
	s.enStreamerTemplate = parseHTMLTemplate(append([]string{"en/streamer.gohtml", "en/trans.gohtml"}, common...)...)
	s.ruStreamerTemplate = parseHTMLTemplate(append([]string{"ru/streamer.gohtml", "ru/trans.gohtml"}, common...)...)
	s.enStreamerNotificationsTemplate = parseHTMLTemplate(append([]string{"en/streamer-notifications.gohtml", "en/trans.gohtml"}, common...)...)
	s.ruStreamerNotificationsTemplate = parseHTMLTemplate(append([]string{"ru/streamer-notifications.gohtml", "ru/trans.gohtml"}, common...)...)
	s.enStreamerChannelTemplate = parseHTMLTemplate(append([]string{"en/streamer-channel.gohtml", "en/trans.gohtml"}, common...)...)
	s.ruStreamerChannelTemplate = parseHTMLTemplate(append([]string{"ru/streamer-channel.gohtml", "ru/trans.gohtml"}, common...)...)

	chic := []string{"common/head.gohtml", "common/header.gohtml", "common/footer.gohtml", "common/cpix.gohtml"}
	s.enChicTemplate = parseHTMLTemplate(append([]string{"common/chic.gohtml", "en/chic.gohtml", "en/trans.gohtml"}, chic...)...)
	s.ruChicTemplate = parseHTMLTemplate(append([]string{"common/chic.gohtml", "ru/chic.gohtml", "ru/trans.gohtml"}, chic...)...)
	s.enPackTemplate = parseHTMLTemplate(append([]string{"en/pack.gohtml", "en/trans.gohtml", "common/twitter.gohtml"}, chic...)...)
	s.ruPackTemplate = parseHTMLTemplate(append([]string{"ru/pack.gohtml", "ru/trans.gohtml", "common/twitter.gohtml"}, chic...)...)
	s.enCodeTemplate = parseHTMLTemplate(append([]string{"en/code.gohtml", "en/trans.gohtml", "common/twitter.gohtml"}, chic...)...)
	s.ruCodeTemplate = parseHTMLTemplate(append([]string{"ru/code.gohtml", "ru/trans.gohtml", "common/twitter.gohtml"}, chic...)...)
}

func (s *server) fillEnabledPacks() {
	packs := make([]sitelib.PackV2, 0, len(s.packs))
	for _, pack := range s.packs {
		if !pack.Disable {
			packs = append(packs, pack)
		}
	}
	s.enabledPacks = packs
}

func main() {
	linf("starting...")
	srv := &server{cfg: sitelib.ReadConfig()}
	srv.packs = sitelib.ParsePacksV2(srv.cfg)
	for _, pack := range srv.packs {
		if pack.ChaturbateIconsScale == nil {
			panic(fmt.Sprintf("pack %s has no chaturbate_icons_scale", pack.Name))
		}
	}
	if len(srv.packs) > 2 {
		srv.packs = append([]sitelib.PackV2{srv.packs[len(srv.packs)-1]}, srv.packs[:len(srv.packs)-1]...)
	}
	srv.fillRawFiles()
	srv.fillTemplates()
	srv.fillEnabledPacks()
	db, err := pgxpool.New(context.Background(), srv.cfg.ConnectionString)
	checkErr(err)
	srv.db = db
	srv.createDatabase()
	fmt.Printf("%d packs loaded, %d icons\n", len(srv.packs), srv.iconsCount())
	ruDomain := "ru." + srv.cfg.BaseDomain
	r := mux.NewRouter().StrictSlash(true)

	bilingualRoute := func(path string, ruHandler, enHandler http.HandlerFunc) {
		if srv.cfg.Lang == "ru" {
			r.Handle(path, srv.measure(handlers.CompressHandler(ruHandler)))
		} else {
			r.Handle(path, srv.measure(handlers.CompressHandler(ruHandler))).Host(ruDomain)
			r.Handle(path, srv.measure(handlers.CompressHandler(enHandler)))
		}
	}

	bilingualRoute("/", srv.ruIndexHandler, srv.enIndexHandler)
	bilingualRoute("/streamer", srv.ruStreamerHandler, srv.enStreamerHandler)
	bilingualRoute("/streamer/notifications", srv.ruStreamerNotificationsHandler, srv.enStreamerNotificationsHandler)
	bilingualRoute("/streamer/channel", srv.ruStreamerChannelHandler, srv.enStreamerChannelHandler)
	bilingualRoute("/chic", srv.ruChicHandler, srv.enChicHandler)
	bilingualRoute("/chic/p/{pack}", srv.ruPackHandler, srv.enPackHandler)
	bilingualRoute("/chic/code/{pack}", srv.ruCodeHandler, srv.enCodeHandler)
	r.Handle("/chic/test/{pack}", srv.measure(handlers.CompressHandler(http.HandlerFunc(srv.testHandler))))
	r.Handle("/chic/like/{pack}", srv.measure(http.HandlerFunc(srv.likeHandler)))

	r.PathPrefix("/icons/").Handler(http.StripPrefix("/icons", cacheControlHandler(http.FileServer(http.Dir("icons")), 120)))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", cacheControlHandler(handlers.CompressHandler(http.FileServer(http.Dir("static"))), 120)))

	r.Handle("/ru", newRedirectSubdHandler("ru", "", http.StatusMovedPermanently))
	r.Handle("/ru.html", newRedirectSubdHandler("ru", "", http.StatusMovedPermanently))
	r.Handle("/streamer-ru", newRedirectSubdHandler("ru", "/streamer", http.StatusMovedPermanently))
	r.Handle("/model.html", http.RedirectHandler("/streamer", http.StatusMovedPermanently))
	r.Handle("/model-ru.html", newRedirectSubdHandler("ru", "/streamer", http.StatusMovedPermanently))

	ln, err := net.Listen("tcp", srv.cfg.ListenAddress)
	checkErr(err)
	linf("listening on %s", ln.Addr())
	checkErr(http.Serve(ln, r))
}
