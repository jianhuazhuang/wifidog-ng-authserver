/*
 * Copyright (C) 2017 Jianhui Zhao <jianhuizhao329@gmail.com>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301
 * USA
 */

package main

import (
    "flag"
    "log"
    "fmt"
    "time"
    "math/rand"
    "strconv"
    "net/http"
    "crypto/md5"
    "encoding/hex"
    "io/ioutil"
    "encoding/json"
    _ "github.com/zhaojh329/wifidog-ng-authserver/statik"
    "github.com/rakyll/statik/fs"
    "github.com/joshbetz/config"
)

var loginPage = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>WiFiDog-ng</title>
    <meta name="viewport" content="width=device-width,minimum-scale=1.0,maximum-scale=1.0,user-scalable=no" />
    <script>
        function load() {
            document.forms[0].action = "/wifidog/login" + window.location.search;
        }
    </script>
</head>
<body onload="load()">
    <div style="position: absolute; top: 50%; left:50%; margin: -150px 0 0 -150px; width: 300px; height: 300px;">
        <h1>Login</h1>
        <form method="POST">
            <button style="width: 300px; min-height: 20px;  padding: 9px 14px; font-size: 20px;" type="submit">Login</button>
        </form>
    </div>
</body>
</html>
`
var portalPage = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>WiFi Portal</title>
    <meta name="viewport" content="width=device-width,minimum-scale=1.0,maximum-scale=1.0,user-scalable=no" />
</head>
<body>
    <h1>Welcome to WiFi Portal</h1>
</body>
</html>
`
func generateToken(mac string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(mac + strconv.FormatFloat(rand.Float64(), 'e', 6, 32)))
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
}

type weixinConfig struct {
    Appid string `json:"appid"`
    Shopid string `json:"shopid"`
    Secretkey string `json:"secretkey"`
}

func main() {
    port := flag.Int("port", 8912, "http service port")
    weixin := flag.Bool("wx", false, "weixin")
    roam := flag.Bool("roam", false, "roam")
    verbose := flag.Bool("v", false, "verbose")
    cert := flag.String("cert", "", "certFile Path")
    key := flag.String("key", "", "keyFile Path")

    flag.Parse()

    rand.Seed(time.Now().Unix())

    c := config.New("weixin.json")
    weixincfg := &weixinConfig{}

    c.Get("appid", &weixincfg.Appid)
    c.Get("shopid", &weixincfg.Shopid)
    c.Get("secretkey", &weixincfg.Secretkey)

    http.HandleFunc("/wifidog/ping", func(w http.ResponseWriter, r *http.Request) {
        if *verbose {
            log.Println("ping", r.URL.RawQuery)    
        }
        
        fmt.Fprintf(w, "Pong")
    })

    statikFS, err := fs.New()
    if err != nil {
        log.Fatal(err)
        return
    }

    staticfs := http.FileServer(statikFS)

    http.HandleFunc("/wifidog/login", func(w http.ResponseWriter, r *http.Request) {
        log.Println("login", r.URL.RawQuery)
        if r.Method == "GET" {
            if *weixin {
                http.Redirect(w, r, "/weixin/login.html?" + r.URL.RawQuery, http.StatusFound)
            } else {
                fmt.Fprintf(w, loginPage)
            }
        } else {
            gw_address := r.URL.Query().Get("gw_address")
            gw_port := r.URL.Query().Get("gw_port")
            mac := r.URL.Query().Get("mac")
            token := generateToken(mac)

            uri := fmt.Sprintf("http://%s:%s/auth?token=%s", gw_address, gw_port, token)
            http.Redirect(w, r, uri, http.StatusFound)
        }
    })

    http.HandleFunc("/wifidog/auth", func(w http.ResponseWriter, r *http.Request) {
        stage := r.URL.Query().Get("stage")

        if stage == "login" {
            log.Println("auth", stage, r.URL.RawQuery)
            fmt.Fprintf(w, "Auth: 1")
        } else if stage == "counters" {
            if *verbose {
                body, _ := ioutil.ReadAll(r.Body)
                r.Body.Close()
                log.Println("auth", stage, r.URL.RawQuery)
                log.Println(string(body))
            }
            fmt.Fprintf(w, "{\"resp\":[]}")
        } else if stage == "roam" {
            log.Println("auth", stage, r.URL.RawQuery)
            if *roam {
                fmt.Fprintf(w, "token=12345678")
            } else {
                fmt.Fprintf(w, "deny")
            }
        } else {
            log.Println("auth", stage, r.URL.RawQuery)
            fmt.Fprintf(w, "OK")
        }
    })

    http.HandleFunc("/wifidog/weixin", func(w http.ResponseWriter, r *http.Request) {
        log.Println("weixin", r.URL.RawQuery)
        gw_address := r.URL.Query().Get("gw_address")
        gw_port := r.URL.Query().Get("gw_port")
        mac := r.URL.Query().Get("mac")
        token := generateToken(mac)

        uri := fmt.Sprintf("http://%s:%s/auth?token=%s", gw_address, gw_port, token)
        http.Redirect(w, r, uri, http.StatusFound)
    })

    http.HandleFunc("/wifidog/portal", func(w http.ResponseWriter, r *http.Request) {
        log.Println("portal", r.URL.RawQuery)
        fmt.Fprintf(w, portalPage)
    })

    http.HandleFunc("/wifidog/weixincfg", func(w http.ResponseWriter, r *http.Request) {
        js, _ := json.Marshal(weixincfg)
        fmt.Fprintf(w, string(js))
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        staticfs.ServeHTTP(w, r)
    })

    log.Println("Listen on: ", *port)
    log.Println("weixin: ", *weixin)
    log.Println("roam: ", *roam)

    if *cert != "" && *key != "" {
        log.Println("Listen on: ", *port, "SSL on")
        log.Fatal(http.ListenAndServeTLS(":" + strconv.Itoa(*port), *cert, *key, nil))
    } else {
        log.Println("Listen on: ", *port, "SSL off")
        log.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
    }
}
