package logWriter

import ()

type Entry struct {
	level   Level       //Level the log entry was logged at: Debug, Info, Warn or Error.
	message interface{} // Message passed to Debug, Info, Warn or Error
	format  string      //format with which logger string would be printed
}

//This method creates and returns new log entry having level and message args.
func NewEntry(level Level, message interface{}) (entry Entry) {
	return Entry{
		level:   level,
		message: message}
}

//This method creates and returns new formatted log entry having level, format and message args.
func NewFormattedEntry(level Level, format string, message interface{}) (entry Entry) {
	return Entry{
		level:   level,
		message: message,
		format:  format}
}
