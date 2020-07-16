package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	customlogger "github.com/mobilejavierg/logs"
	"github.com/mobilejavierg/ofiles"
)

//Configuration configuracion
type Configuration struct {
	Puerto         string
	DirectorioRaiz string
}

var _configuration Configuration

//LLogger log
var LLogger *customlogger.Logger

func main() {

	LLogger = customlogger.GetInstance("alphaFileServer", "main ")
	LLogger.Println("#### Inicia servicio GoFileServer ####\r")

	//lee archivo de configuracion
	getConfiguracion()

	//si no existe el directotio raiz
	if _, err := os.Stat(_configuration.DirectorioRaiz); os.IsNotExist(err) {
		// creo el DirectorioRaiz
		os.MkdirAll(_configuration.DirectorioRaiz, os.ModePerm)
	}

	http.HandleFunc("/upload", upload)
	errHTTP := http.ListenAndServe(":"+_configuration.Puerto, nil)

	if errHTTP != nil {
		LLogger.Println(errHTTP)
		fmt.Println(errHTTP)
		return
	}

}

func getConfiguracion() {

	//////////////////////////////////////////////////////////////////////
	// leer archivo configuracion
	LLogger.Println("levantando configuracion buscando alphafileserver.json")

	//filename is the path to the json config file
	file, err := os.Open("alphafileserver.json")

	if err != nil {
		LLogger.Println("ERROR open gofileserver.json " + err.Error())
		fmt.Println(err)
		os.Exit(0)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&_configuration)
	if err != nil {
		LLogger.Println("ERROR decode json " + err.Error())
		fmt.Println(err)
	}

	fmt.Println("escuchado: " + _configuration.Puerto)
	fmt.Println("directorio Raiz: " + _configuration.DirectorioRaiz)

	// fin archivo de configuracion
	//////////////////////////////////////////////////////////////////////
}

// upload logic
func upload(w http.ResponseWriter, r *http.Request) {

	var _destino string
	//	logger := customlogger.GetInstance("goFileServer", "Main UpLoad ")
	LLogger.Println("\rconneccion desde: " + r.RemoteAddr + " metodo: " + r.Method + "\r")

	if r.Method == "GET" {
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))
		fmt.Fprintf(w, token)

		/*
			t, _ := template.ParseFiles("upload.gtpl")
			t.Execute(w, token)
			t.WriteString(token)
		*/
	} else {

		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			LLogger.Println(err.Error() + "\r")
			fmt.Println(err)
			return
		}
		defer file.Close()

		_hashorigen := r.FormValue("hash")
		_destino = r.FormValue("destino")
		_destino = filepath.Join(_configuration.DirectorioRaiz, _destino, "\\")

		//si no existe el directotio raiz
		if _, err := os.Stat(_destino); os.IsNotExist(err) {
			// creo el DirectorioRaiz
			os.MkdirAll(_destino, os.ModePerm)
		}

		LLogger.Println("archivo recibido: " + handler.Filename + " tamaño: " + strconv.FormatInt((handler.Size/1024), 10) + " KB  hash: " + r.FormValue("hash") + "\r")

		f, err := os.OpenFile(_destino+"\\"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		LLogger.Println("generando " + _destino + "\\" + handler.Filename)

		if err != nil {
			LLogger.Println(err.Error() + "\r")
			fmt.Println(err)
			return
		}
		defer f.Close()
		io.Copy(f, file)

		//gethash
		_hashDestino, _ := ofiles.Hashfilemd5(filepath.Join(_destino + "\\" + handler.Filename))

		if _hashorigen != _hashDestino {
			LLogger.Println(w, "%v", "Error Hash: "+_hashorigen+" -> "+_hashDestino)
			fmt.Fprintf(w, "%v", "Error Hash: "+_hashorigen+" -> "+_hashDestino)
		} else {
			fmt.Fprintf(w, "%v", "OK")
			LLogger.Println("OK " + f.Name() + " hash: " + _hashorigen + "\r")
		}

	}
}
