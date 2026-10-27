// Copyright (c) 2026, The WaveReality Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wavesim

import (
	"slices"
	"sync"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

// releaseWait is the longest time that a sim whose GUI is destroyed is
// waited for to stop running before it is released anyway.
const releaseWait = 5 * time.Second

// releasing has the sims whose GUI is destroyed and that are being
// released once they stop running, which [Embed] waits for before it
// makes the next sim, as the state and GPU of the sims are global.
var releasing struct {
	sync.Mutex
	sims []chan struct{} // closed when released
}

// simFrame is the frame that [Embed] puts the GUI of a sim in, which stops
// and releases the sim when it is destroyed, as when the page of a content
// app that shows it is left.
type simFrame struct { //core:no-new
	core.Frame

	// sim is the sim.
	sim *Sim

	// released is closed once the sim is released.
	released chan struct{}
}

func (sf *simFrame) Init() {
	sf.Frame.Init()
	sf.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
}

func (sf *simFrame) Destroy() {
	if sf.released != nil {
		sf.release()
	}
	sf.Frame.Destroy()
}

// release stops the sim, and then releases it: now if it is not running,
// and otherwise once it stops, in another goroutine, so that the event
// loop, which its GUI updates need, is not blocked.
func (sf *simFrame) release() {
	released := sf.released
	sf.released = nil
	ss := sf.sim
	done := func() {
		ss.Destroy()
		close(released)
	}
	if ss == nil || !ss.GUI.IsRunning() {
		if ss != nil {
			done()
		} else {
			close(released)
		}
		return
	}
	ss.GUI.SetStopNow()
	releasing.Lock()
	releasing.sims = append(releasing.sims, released)
	releasing.Unlock()
	go func() {
		for start := time.Now(); ss.GUI.IsRunning() && time.Since(start) < releaseWait; {
			time.Sleep(10 * time.Millisecond)
		}
		done()
		releasing.Lock()
		releasing.sims = slices.DeleteFunc(releasing.sims, func(c chan struct{}) bool { return c == released })
		releasing.Unlock()
	}()
}

// pendingReleases returns the channels of the sims that are being released.
func pendingReleases() []chan struct{} {
	releasing.Lock()
	defer releasing.Unlock()
	return slices.Clone(releasing.sims)
}

// Destroy releases the GPU resources of the sim, when its GUI is destroyed
// (see [Embed]), so that the next sim initializes the GPU for itself.
func (ss *Sim) Destroy() {
	GPURelease()
}
