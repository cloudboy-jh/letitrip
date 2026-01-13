package organize

type ChannelReporter struct {
	ch chan<- ProgressEvent
}

func NewChannelReporter(ch chan<- ProgressEvent) ProgressReporter {
	return ChannelReporter{ch: ch}
}

func (r ChannelReporter) Report(event ProgressEvent) {
	if r.ch == nil {
		return
	}
	r.ch <- event
}
