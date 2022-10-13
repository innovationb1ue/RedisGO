package util

type Pump struct {
	Out   chan int
	In    map[int]<-chan struct{}
	index int
}

func NewPump() *Pump {
	return &Pump{
		Out:   make(chan int, 1),
		In:    make(map[int]<-chan struct{}, 0),
		index: -1,
	}
}

func (p *Pump) AddIn(in <-chan struct{}) {
	p.index++
	p.In[p.index] = in
}

// RunForward blocks and forwards all inbound channels to a single outbound channel for exactly 1 message.
func (p *Pump) RunForward() {
	// chan to stop other listening goroutines
	msgSent := make(chan struct{})
	for idx, in := range p.In {
		in := in
		idx := idx
		go func(in <-chan struct{}, idx int) {
			select {
			case <-msgSent:
				return
			case <-in:
				// output the index of the available list
				p.Out <- idx
				close(msgSent)
				return
			}
		}(in, idx)
	}
}
