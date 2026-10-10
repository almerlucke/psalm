package psalm

import (
	"sync"
	"testing"
)

func TestWithin(t *testing.T) {
	host := New(nil)
	sub := host.NewSubWith(CurrentStackConfig())
	if CurrentEnv() != host.Env() {
		t.Fatal("making a patch to build elsewhere should not change the current environment")
	}
	var inSub, inHost []*Module
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // a song built in the background
		defer wg.Done()
		sub.Within(func() {
			for range 300 {
				inSub = append(inSub, NewControl())
			}
		})
	}()
	go func() { // meanwhile another goroutine builds for the patch around it
		defer wg.Done()
		for range 300 {
			inHost = append(inHost, NewControl())
		}
	}()
	wg.Wait()
	for _, m := range inSub {
		if m.Env() != sub.Env() {
			t.Fatal("a module built within the song should belong to the song")
		}
	}
	for _, m := range inHost {
		if m.Env() != host.Env() {
			t.Fatal("a module built elsewhere meanwhile should belong to its own patch")
		}
	}
	sub.Within(func() {
		inner := host.NewSubWith(CurrentStackConfig())
		inner.Within(func() {
			if NewControl().Env() != inner.Env() {
				t.Fatal("within within: the inner patch")
			}
		})
		if NewControl().Env() != sub.Env() {
			t.Fatal("after the inner one the outer patch again")
		}
	})
	if NewControl().Env() != host.Env() {
		t.Fatal("after Within the patch of the program again")
	}
}

func TestNewSubIsCurrent(t *testing.T) {
	host := New(nil)
	sub := host.NewSub()
	if CurrentEnv() != sub.Env() || NewControl().Env() != sub.Env() {
		t.Fatal("NewSub should make its environment current, like New")
	}
	SetCurrentEnv(host.Env())
}

func TestConfigFromGoroutines(t *testing.T) {
	New(nil)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				PushConfig(Config{SampleRate: 44100, BufferSize: 64, BPM: 100, PPQ: 96})
				_ = CurrentStackConfig()
				PopConfig()
			}
		}()
	}
	wg.Wait()
}
