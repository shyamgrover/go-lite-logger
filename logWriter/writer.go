package logWriter

import (
	"github.com/shyamgrover/go-lite-logger/utils"
	"log"
	"os"
	"sync"
	"time"
)

type Worker struct {
	once          sync.Once           //for singleton operations
	fileRoot      *os.File            //file to which log entries would be written.
	buffer        []byte              //temporarily keeps log entries before writing to file.
	position      int                 //position to maintain upto which index in buffer data is written to disk.
	Info          *log.Logger         //Info log handle.
	Warning       *log.Logger         //Warning log handle.
	Error         *log.Logger         //Error log handle.
	Debug         *log.Logger         //Debug log handle.
	channel       <-chan Entry        //Channel that will receive log entries.
	lock          sync.Mutex          //lock to synchronize between capacity and timer based flush to file.
	ticker        *time.Ticker        //timer
	quitTimer     chan struct{}       //stop timer channel
	done          chan struct{}       //stop worker channel
	errorCallback utils.ErrorFunction //user defined error callback function..to be invoked in case of error
}

//default flush timer repeat interval in seconds.
const defaultFlushLogsTimerInterval = 10

//buffer's default capacity
const capacity = 32768

//default flag for log entries
const defaultLogFlag = log.LstdFlags | log.Lmicroseconds | log.Lshortfile

//This returns a new instance of a worker. It takes file, channel(in read only mode) and callback as
// arguments and returns a new worker. The returned worker reads continuously from channel and fills its buffer.
// This buffer is flushed on to the disk to the given file. Flushing is of 2 types:
// Capacity Based Flushing: There is some default buffer capacity defined. When the buffer reaches its
// capacity, it flushes the entries from buffer on to the file.
// Timer Based Flushing: A timer job is initiated when new worker is instantiated and it runs periodically
// to flush the entries from the buffer on to the file. This is required when logging on to a channel is
// not too frequent. In this case buffer will be lesser than its default capacity and will never flush
// to the disk. So timer job will run and will flush the log entries to the file.
func NewWorker(file *os.File, channel <-chan Entry, errorCallback utils.ErrorFunction) (worker *Worker) {
	newWorker := Worker{
		fileRoot:      file,
		buffer:        make([]byte, capacity),
		channel:       channel,
		ticker:        time.NewTicker(defaultFlushLogsTimerInterval * time.Second),
		quitTimer:     make(chan struct{}),
		done:          make(chan struct{}),
		errorCallback: errorCallback,
	}
	newWorker.init()
	return &newWorker
}

//This method will initialize the worker by creating different log handles say; Info, Error, Warning and
// Debug. Also it will start a timer job(new go-routine) that would run periodically to flush the
// buffer(containing log entries) to the disk.
func (w *Worker) init() {
	w.createLogHandles()
	w.doTimerJob()
}

//This method returns if file(to which log entries are to be written) exists on the disk or not.
func (w *Worker) fileExists() bool {
	fileName := w.fileRoot.Name()
	if _, err := os.Stat(fileName); err == nil {
		return true
	} else {
		return false
	}
}

//This is the overridden implementation of io.Writer interface. This method writes log entry on worker's
// buffer. The method first checks if (previous buffer capacity + new log entry length) > buffer's capacity,
// then it calls the save method on writer to save buffered entries and if save is successful, it will
// copy new event data(received as argument to Write method) to the buffer. And will update the position
// accordingly. If there is some error while writing buffer to file, then, provided callback method will be executed.
func (w *Worker) Write(data []byte) (n int, err error) {
	length := len(data)
	w.lock.Lock()
	if (length + w.position) > capacity {
		n, err = w.save()
		if err != nil {
			w.errorCallback()
			return n, err
		}
	}
	copy(w.buffer[w.position:], data)
	w.position += length
	w.lock.Unlock()
	return n, err
}

//This method writes the buffered log entries to the file. This copies data from position 0 to buffer's
// current length and after writing to file, if save is successful, it sets the buffer position to 0 and
// if there is some error while writing to file, it will return error to its caller.
func (w *Worker) save() (n int, err error) {
	if w.position == 0 {
		return 0, nil
	}
	if w.fileExists() {
		n, err = w.fileRoot.Write(w.buffer[0:w.position])
		if err == nil {
			w.position = 0
		}
	} else {
		w.errorCallback()
	}
	return n, err
}

//Worker spends most of the time in this method. This method is called as a separate goroutine after
// instantiating the worker. The method checks in an infinite loop if worker is closed or not. If closed, it returns
// from the method and if not, reads continuously from channel and fills its buffer.
func (w *Worker) Work() {
	for {
		select {
		case <-w.done:
			return
		default:
			event := <-w.channel
			w.writeToBuffer(event)
		}
	}
}

//This method checks entry's log level and format and calls appropriate handle to write it to the buffer.
func (w *Worker) writeToBuffer(event Entry) {
	switch event.level {
	case WarnLevel:
		if len(event.format) > 0 {
			w.Warning.Printf(event.format, event.message)
		} else {
			w.Warning.Println(event.message)
		}
	case InfoLevel:
		if len(event.format) > 0 {
			w.Info.Printf(event.format, event.message)
		} else {
			w.Info.Println(event.message)
		}
	case DebugLevel:
		if len(event.format) > 0 {
			w.Debug.Printf(event.format, event.message)
		} else {
			w.Debug.Println(event.message)
		}
	case ErrorLevel:
		if len(event.format) > 0 {
			w.Error.Printf(event.format, event.message)
		} else {
			w.Error.Println(event.message)
		}
	}
}

//This method is used to close the worker resources. First it will stop the timer by closing quitTimer channel,
// then it stops the worker by closing done channel. Then it calls save to flush buffer entries to file. Then it loops
// over the channel length(if there were some entries remaining on channel) and writes to buffer. Now, if the capacity
// is full in between, capacity based flushing will run automatically and finally if the buffer content is less than
// its capacity, the after loop exit, save method will be called to flush off the buffer to file. This way all
// buffer data and channel entries are flushed on to disk on worker close.
func (w *Worker) CloseWorker() {
	w.once.Do(func() {
		close(w.done)
		close(w.quitTimer)

		w.lock.Lock()
		w.save()
		w.lock.Unlock()

		length := len(w.channel)
		for i := 0; i < length; i++ {
			event := <-w.channel
			w.writeToBuffer(event)
		}
		w.lock.Lock()
		w.save()
		w.lock.Unlock()
	})
}

//This method starts a timer job that is initiated when new worker is instantiated and it runs periodically
// to flush the entries from the buffer on to the file. This is required when logging on to a channel is
// not too frequent. In this case buffer will be lesser than its default capacity and will never flush
// to the disk. So timer job will run and will flush the log entries to the file.
func (w *Worker) doTimerJob() {
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.lock.Lock()
				_, err := w.save()
				if err != nil {
					w.errorCallback()
				}
				w.lock.Unlock()
			case <-w.quitTimer:
				w.ticker.Stop()
				return
			}
		}
	}()
}

//This method creates different level based log handles and their output is set to the worker.
//Worker is implementing io.Writer interface. These handles write to the worker's buffer.
func (w *Worker) createLogHandles() {
	w.Info = log.New(w,
		"[INFO]  ",
		defaultLogFlag)

	w.Warning = log.New(w,
		"[WARN]  ",
		defaultLogFlag)

	w.Error = log.New(w,
		"[ERROR] ",
		defaultLogFlag)

	w.Debug = log.New(w,
		"[DEBUG] ",
		defaultLogFlag)
}
