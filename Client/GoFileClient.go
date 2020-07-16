package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	customloggerC "github.com/mobilejavierg/logs"
	"github.com/mobilejavierg/ofiles"
	//    "github.com/fsnotify/fsnotify"
)

//LLoggerC log
var LLoggerC *customloggerC.Logger

func existeArchivo(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}

	/*quise poner todo esto en una linea, pero no lo logre :( */
	existe := true
	if err != nil {
		LLoggerC.Println("existeArchivo " + name + " - " + err.Error())
		existe = false
	}

	return existe, err
}

func confirmarArchivo(_directorio string, _archivoOK string) {

	archivoNuevo := ""

	if _directorio != "" {

		i := strings.Index(_archivoOK, strings.Replace(_directorio, "\\\\", "\\", -1))
		if i > -1 {
			archivoNuevo = _archivoOK + ".ok"
		} else {
			archivoNuevo = filepath.Join(_directorio, _archivoOK+".ok")
		}
	} else {
		archivoNuevo = _archivoOK + ".ok"
	}

	newFile, err := os.Create(archivoNuevo)
	if err != nil {
		LLoggerC.Println("creando archivo Ok " + archivoNuevo + " - " + err.Error())
		fmt.Println("creando archivo Ok " + archivoNuevo + " - " + err.Error())
	}
	newFile.Close()
}

func procesarArchivos(_directorio string, _url string, _extension string, _destino string, _minutespan int64) {

	files, err := ioutil.ReadDir(_directorio)

	if err != nil {
		LLoggerC.Println("error accediendo a " + _directorio + " - " + err.Error())
		fmt.Println("error accediendo a " + _directorio + " - " + err.Error())
	}

	// utc life
	timeloc, _ := time.LoadLocation("UTC")
	nowTime := time.Now().In(timeloc)

	//	LLoggerC.Println("error accediendo a " + _directorio + " - " + err.Error())

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), _extension) {

			filename := f.Name()

			existe, err := existeArchivo(filepath.Join(_directorio, filename+".ok"))

			if err != nil {
				fmt.Println("Error:" + filepath.Join(_directorio, filename+".ok") + " " + err.Error())
				//				LLoggerC.Println("Error:" + filepath.Join(_directorio, filename+".ok") + " " + err.Error())
			}

			if existe {
				fmt.Println("archivo " + filepath.Join(_directorio, filename) + " ya enviado previamente")
				LLoggerC.Println("archivo " + filepath.Join(_directorio, filename) + " ya enviado previamente")
			} else {

				filecurrTime := f.ModTime()
				diff := int64(nowTime.Sub(filecurrTime).Minutes())

				if (_minutespan == 0) || (_minutespan > 0 && (diff > _minutespan)) {
					fmt.Println(filepath.Join(_directorio, filename))
					if _minutespan > 0 {
						fmt.Println(" " + strconv.FormatInt(diff, 10) + " minutos de vida")
					}
					LLoggerC.Println("enviando " + filename)
					postFile(_directorio, filename, _url, _destino, "")
				}
			}
		}
	}
}

func postFile(directorio string, _filename string, targetURL string, destino string, archivoFullpath string) error {

	var _archivoProcesar string

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	if archivoFullpath != "" {
		_filename = filepath.Base(archivoFullpath)
	}
	// this step is very important

	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", _filename)
	if err != nil {
		LLoggerC.Println("error writing to buffer -- uploadfile" + _filename + " - " + err.Error())
		fmt.Println("error writing to buffer " + err.Error())
		os.Exit(1)
	}

	if archivoFullpath == "" {
		_archivoProcesar = filepath.Join(directorio, _filename)
	} else {
		_archivoProcesar = archivoFullpath
	}

	// open file handle

	fmt.Println("open file handle")

	fh, err := os.Open(_archivoProcesar)
	if err != nil {
		LLoggerC.Println("error opening file -- open file handle" + _archivoProcesar + " - " + err.Error())
		fmt.Println("error opening file" + _archivoProcesar + " - " + err.Error())
		os.Exit(1)
	}
	defer fh.Close()

	//gethash
	_hash, _ := ofiles.Hashfilemd5(_archivoProcesar)

	//iocopy
	fmt.Println("iocopy")

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		fmt.Println("error io.copy " + " - " + err.Error())
		LLoggerC.Println("error io.copy " + " - " + err.Error())
		os.Exit(1)
	}

	bodyWriter.WriteField("hash", _hash)
	bodyWriter.WriteField("destino", destino)

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetURL, contentType, bodyBuf)

	if err != nil {
		fmt.Println("error post " + targetURL + " - " + err.Error())
		LLoggerC.Println("error post " + targetURL + " - " + err.Error())
		os.Exit(1)
	}

	defer resp.Body.Close()
	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error ioutil.ReadAll " + " - " + err.Error())
		LLoggerC.Println("error ioutil.ReadAll " + " - " + err.Error())
		os.Exit(1)
	}

	if string(respbody) == "OK" {
		LLoggerC.Println("enviado ok \r")
		confirmarArchivo(directorio, _archivoProcesar)
	}

	return nil
	//fmt.Println(resp.Status)
	//fmt.Println(string(bodyBuf.String()))
	//fmt.Println(string(contentType))

}

func main() {

	var _Servidor string
	var _puerto string
	var _Directorio string
	var _archivo string
	var _Extensionarchivo string
	var _destino string
	var _lifeSpan int64

	flag.StringVar(&_Servidor, "ip", "", "direccion servidor (192.168.2.90)") // (http://192.168.2.90:1678/upload)")
	flag.StringVar(&_puerto, "port", "1678", "puerto del servidor (port=9999)")

	flag.StringVar(&_Directorio, "path", "", "directorio origen (path=\"c:\\archivos ret\\\\\" doble barra al final)")
	flag.StringVar(&_archivo, "archivo", "", "fullpath + archivo archivo=c:\\carpeta\\archivo.ext, si envia este paramtro se ingora [PATH] y [EXT]")

	flag.StringVar(&_Extensionarchivo, "ext", "", "extension del archivo a enviar (ext=ret)")
	flag.StringVar(&_destino, "dest", "", "destino en servidor (ext=\\678\\20276862465\\)")
	flag.Int64Var(&_lifeSpan, "span", 0, "_lifeSpam, tiempo de vida para copiar, por defecto 0 (deshabilitado)")

	flag.Parse()

	if _Directorio == "" && _archivo == "" {
		flag.Usage()
		os.Exit(0)
	}

	if _Directorio != "" && (_Servidor == "" || _Extensionarchivo == "" || _puerto == "") {
		flag.Usage()
		os.Exit(0)
	}

	if _archivo != "" && (_Servidor == "" || _puerto == "") {
		flag.Usage()
		os.Exit(0)
	}

	if !strings.HasSuffix(_Directorio, "\\") {
		_Directorio = strings.TrimSpace(_Directorio) + "\\"
	}

	LLoggerC = customloggerC.GetInstance("gofileclient", "client ")

	fmt.Println(_Directorio)
	fmt.Println("Alpha 2000 Soluciones Informaticas")
	fmt.Println("Servidor: " + _Servidor + ":" + _puerto)

	_urlServidor := "http://" + _Servidor + ":" + _puerto + "/upload"

	LLoggerC.Println(" Iniciando copia desde " + _Directorio + " al Servidor: " + _Servidor + ":" + _puerto)

	if _archivo == "" {
		procesarArchivos(_Directorio, _urlServidor, _Extensionarchivo, _destino, _lifeSpan)
	} else {
		fmt.Println(_archivo)
		postFile("", "", _urlServidor, _destino, _archivo)
	}

}

/*

var _hash string

watcher, err := fsnotify.NewWatcher()
if err != nil {
    fmt.Println(err)
}
defer watcher.Close()

done := make(chan bool)
go func() {
    for {
        select {
        case event := <-watcher.Events:

            fmt.Println("event:", event)
            if event.Op&fsnotify.Write == fsnotify.Write {

//                    _hash, _ = ofiles.Hashfilemd5(strings.Replace(event.Name,'d','d',-1))

                fmt.Println("modified file:", event.Name, " hash: " , _hash)
            }
            if event.Op&fsnotify.Create == fsnotify.Create {
                fmt.Println("created file:", event.Name)
            }
            _hash, _ = ofiles.Hashfilemd5(event.Name)
            fmt.Println(_hash)

        case err := <-watcher.Errors:
            fmt.Println("error:", err)
        }
    }
}()

err = watcher.Add("C:/_cryptoAPI/")
if err != nil {
    fmt.Println(err)
}
<-done
*/
