package main
// Version: 0.5.1

import (
	"fmt"
	"net/http"
	"time"
	"strconv"
	"runtime"
	"strings"
	"os"
)

const JOB_TIMEOUT = 60*60
const JOB_TRIGGER_TIMEOUT = 60*60

type ES struct{}

type Control struct {
	msg	string
	err	error
}

type Job struct {
	blocker chan ES
	timeout chan ES
	abort		chan ES
	mode 		int
	sleep 	int
	name		string
	start		time.Time
}

type Harmonizator struct {
	jobs	map[string]Job
	control chan Control
	mutex chan ES
}

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

func (hrm Harmonizator) Lock() {
	<- hrm.mutex
}

func (hrm Harmonizator) Unlock() {
	hrm.mutex <- ES{}
}

func master_timeout(hrm Harmonizator, job Job) {
	select {
		case <-time.After(time.Second * time.Duration(job.sleep)):
			close(job.blocker)
			break
		case <-time.After(JOB_TIMEOUT*time.Second):
			close(job.timeout)
			break
		case <- job.blocker:
	}
	hrm.Lock()
	delete(hrm.jobs,job.name)
	hrm.Unlock()
}

func master_trigger(hrm Harmonizator,job Job) {
	select {
		case <-time.After(JOB_TRIGGER_TIMEOUT * time.Second):
			close(job.timeout)
			break
		case <- job.blocker:
		case <- job.abort:
	}
	hrm.Lock()
	delete(hrm.jobs,job.name)
	hrm.Unlock()
}

// Returns name, option, mode and timeout
func parseUri(rawuri string) (string,string,int,int) {
	uri:=strings.SplitN(rawuri[1:],":",2)
	name:=uri[0]
	// 1 trigger, 2 timeout
	mode:=int(1)
	sleeptime:=int(0)
	opt:=""
	if len(uri) > 1 {
		if st,err:=strconv.ParseUint(uri[1],10,64); err == nil {
				mode=2
				if st >= JOB_TIMEOUT {
					st=JOB_TIMEOUT-1
				}
				sleeptime=int(st)
				name=fmt.Sprintf("%s:%d",name,sleeptime)
		} else {
			opt=uri[1]
		}
	}
	return name,opt,mode,sleeptime
}

func timeTrack(start time.Time,name string) {
	fmt.Printf("'%s' TOOK %s\n",name,time.Since(start))
}

func (hrm Harmonizator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name,opt,mode,sleeptime:=parseUri(r.RequestURI)
	remote:=r.Header.Get("X-Remote-Addr")
	if remote == "" {
		ip:=strings.Split(r.RemoteAddr,":")
		remote=ip[0]
	}
	if name == "" {
		//indexPage(w)
		return
	}
	hrm.Lock()
	job,ok:=hrm.jobs[name]
	if ! ok {
		job=Job{ name: name, sleep: sleeptime, mode: mode, start: time.Now()}
		job.blocker=make(chan ES)
		job.timeout=make(chan ES)
		job.abort=make(chan ES)
		hrm.jobs[name]=job
		if sleeptime > 0 {
			go master_timeout(hrm,job)
		} else {
			go master_trigger(hrm,job)
		}
		fmt.Printf("JOB START '%s' MODE %v TIMEOUT %v %v %v\n", name, mode, sleeptime,r.Host,remote)
	} else {
		job=hrm.jobs[name]
		if sleeptime > 0 {
			t:=job.sleep-int(time.Now().Sub(job.start).Seconds())
			fmt.Printf("QUEUE JOB: %v (%v s AGO / LEFT %v s) %v %v\n",job.name,int(time.Since(job.start).Seconds()),t,r.Host,remote)
		} else {
			fmt.Printf("QUEUE JOB: %v (STARTED %v s AGO) %v %v\n",job.name,int(time.Since(job.start).Seconds()),r.Host,remote)
		}
	}
	hrm.Unlock()
	if mode == 1 && ( opt == "start" || opt == "run" ) {
		fmt.Printf("TRIGGER JOB: %v %v %v\n",name,r.Host,remote)
		close(job.blocker)
		return
	} else if mode == 1 && ( opt == "abort" || opt == "cancel" ) {
		fmt.Printf("CANCEL JOB: %v %v %v\n",name,r.Host,remote)
		close(job.abort)
		return
	}
	abort:=false
	timeout:=false
	select {
		case <-job.blocker:
			fmt.Printf("JOB UNLOCKED: %s %v %v\n",name,r.Host,remote)
		case <-job.timeout:
			fmt.Printf("JOB TIMEOUT: %s %v %v\n",name,r.Host,remote)
      abort=true
			timeout=true
    case <-r.Context().Done():
			fmt.Printf("CLIENT ABORT: %s %v %v\n",name,r.Host,remote)
      abort=true
		case <-job.abort:
			fmt.Printf("JOB ABORT: %s %v %v\n",name,r.Host,remote)
			abort=true
  }
	if timeout || abort {
		w.WriteHeader(404)
	}
	return
}

func main() {
	hrm:=Harmonizator{}
	hrm.jobs=make(map[string]Job)
	hrm.mutex=make(chan ES,1)
	hrm.control=make(chan Control,1)
	hrm.Unlock()

	go func() {
		err:=http.ListenAndServe(getenv("HARMONIZATOR_LISTEN",":80"),hrm)
		hrm.control <- Control{ msg: "server-abort", err: err }
	}()

	loop:
	for {
			select {
				case cntrl,isopen:=<-hrm.control:
					if !isopen {
							fmt.Printf("SERVER SHUTDOWN\n")
							break loop
					}
					switch cntrl.msg {
						case "server-abort":
							fmt.Printf("SERVER-ABORT: %v\n",cntrl.err)
							break loop
					}
				case <-time.After(30 * time.Second):
					fmt.Printf("- PING %v -\n",runtime.NumGoroutine())
			}
	}
}
