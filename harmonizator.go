package main
// alpha version 
// it's broken (some race conditions found)
// but it's working 

import (
  "log"
	"net/http"
	"os"
  "os/signal"
  "syscall"
  "time"
  "regexp"
  "sync"
  "strconv"
  "runtime"
)

var lock sync.Mutex
var keys = make(map[string]*sync.Mutex)
var be_quiet bool = false

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

func logmer(r *http.Request, info string) {
  if ! be_quiet {
    lock.Lock()
    log.Printf("%v %v %v/%v %v %v",r.Host,r.RemoteAddr,runtime.NumGoroutine(),len(keys),r.RequestURI,info)
    lock.Unlock()
  }
}

func serve(w http.ResponseWriter, r *http.Request) {
  uri:=r.RequestURI
  if uri == "/info" {
    logmer(r,"200 INFO")
    return
  }
  re,_ := regexp.Compile(`^/([^:]+):([0-9]{1,4})$`)
  if re.MatchString(uri) {
    param:=re.FindStringSubmatch(uri)
    tm, err := strconv.Atoi(param[2])
    if err != nil {
      logmer(r,"500")
      w.WriteHeader(500)
      return
	  } 
    if tm < 1 || tm > 3600 {
      logmer(r,"403")
      w.WriteHeader(403)
      return
    }
    lock.Lock()
    if m,ok:=keys[uri]; ok {
      lock.Unlock()
      logmer(r,"102")
      m.Lock()
      m.Unlock()
      logmer(r,"100")
    } else {
      keys[uri]=new(sync.Mutex)
      keys[uri].Lock()
      lock.Unlock()
      logmer(r,"201 WAIT")
      abort:=false
      select {
        case <-time.After(time.Duration(tm) * time.Second):
        case <-r.Context().Done():
            abort=true
      }
      lock.Lock()
      keys[uri].Unlock()
      // I'm counting on GC here to cleanup mutexes
      delete(keys,uri)
      lock.Unlock()
      if abort {
        logmer(r,"200 CANCEL")
      } else {
        logmer(r,"200 DONE")
      }
    }
  } else {
    logmer(r,"404")
    w.WriteHeader(404)
  }
}

func shutdown_me(shutChan chan os.Signal) {
  <- shutChan
  os.Exit(0)
}

func main() {
  if getenv("HARMONIZATOR_QUIET","") != "" {
    be_quiet = true
  }
  shutChan := make(chan os.Signal, 1)
  signal.Notify(shutChan, syscall.SIGTERM, syscall.SIGINT,syscall.SIGQUIT)
  go shutdown_me(shutChan)
	mux := http.NewServeMux()
	mux.HandleFunc("/", serve)
  err := http.ListenAndServe(getenv("HARMONIZATOR_LISTEN",":80"), mux)
  if err != nil {
    if ! be_quiet {
      log.Printf("- - 500 ERROR %s", err)
    }
    os.Exit(1)
  }
}
