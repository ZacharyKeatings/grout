package utils

import (
	"grout/romm"
	"sync"
	"sync/atomic"
	"time"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	icons "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool/constants"
)

const syncIconDelay = 400 * time.Millisecond

type SyncIssue struct {
	Sync          SaveSync
	NeedsEmulator bool
	ErrorMessage  string
}

type AutoSync struct {
	host       romm.Host
	icon       *gaba.DynamicStatusBarIcon
	running    atomic.Bool
	done       chan struct{}
	hasIssues  atomic.Bool
	showButton atomic.Bool
	issues     []SyncIssue
	issuesMu   sync.Mutex
}

func NewAutoSync(host romm.Host) *AutoSync {
	return &AutoSync{
		host: host,
		icon: gaba.NewDynamicStatusBarIcon(""),
		done: make(chan struct{}),
	}
}

func (a *AutoSync) Icon() gaba.StatusBarIcon {
	return gaba.StatusBarIcon{
		Dynamic: a.icon,
	}
}

func (a *AutoSync) Start() {
	a.running.Store(true)
	a.done = make(chan struct{}) // Reinitialize channel for reuse
	go a.run()
}

func (a *AutoSync) IsRunning() bool {
	return a.running.Load()
}

func (a *AutoSync) Wait() {
	<-a.done
}

func (a *AutoSync) HasIssues() bool {
	return a.hasIssues.Load()
}

func (a *AutoSync) ShowButton() *atomic.Bool {
	return &a.showButton
}

func (a *AutoSync) GetIssues() []SyncIssue {
	a.issuesMu.Lock()
	defer a.issuesMu.Unlock()
	result := make([]SyncIssue, len(a.issues))
	copy(result, a.issues)
	return result
}

func (a *AutoSync) ClearIssues() {
	a.issuesMu.Lock()
	defer a.issuesMu.Unlock()
	a.issues = nil
	a.hasIssues.Store(false)
	a.showButton.Store(false)
	a.icon.SetText("")
}

func (a *AutoSync) MarkComplete() {
	a.issuesMu.Lock()
	defer a.issuesMu.Unlock()
	a.issues = nil
	a.hasIssues.Store(false)
	a.showButton.Store(false)
	a.icon.SetText(icons.CloudCheck)
}

func (a *AutoSync) addIssue(issue SyncIssue) {
	a.issuesMu.Lock()
	defer a.issuesMu.Unlock()
	a.issues = append(a.issues, issue)
}

func (a *AutoSync) RemoveIssue(issue SyncIssue) {
	a.issuesMu.Lock()
	defer a.issuesMu.Unlock()
	for i, existing := range a.issues {
		if existing.Sync.GameBase == issue.Sync.GameBase && existing.Sync.Slug == issue.Sync.Slug {
			a.issues = append(a.issues[:i], a.issues[i+1:]...)
			break
		}
	}
	if len(a.issues) == 0 {
		a.hasIssues.Store(false)
		a.showButton.Store(false)
	}
}

func (a *AutoSync) Host() romm.Host {
	return a.host
}

func (a *AutoSync) run() {
	logger := gaba.GetLogger()
	defer func() {
		a.running.Store(false)
		close(a.done)
	}()

	a.icon.SetText(icons.CloudRefresh)
	time.Sleep(syncIconDelay)
	logger.Debug("AutoSync: Starting save sync scan")

	syncs, _, err := FindSaveSyncs(a.host)
	if err != nil {
		logger.Error("AutoSync: Failed to find save syncs", "error", err)
		a.icon.SetText(icons.CloudAlert)
		a.hasIssues.Store(true)
		a.showButton.Store(true)
		return
	}

	logger.Debug("AutoSync: Found syncs", "count", len(syncs))

	if len(syncs) == 0 {
		a.icon.SetText(icons.CloudCheck)
		logger.Debug("AutoSync: No syncs needed")
		return
	}

	hadError := false
	hadSkipped := false

	for i := range syncs {
		s := &syncs[i]

		// Skip syncs that need emulator selection - can't prompt in background
		if s.NeedsEmulatorSelection() {
			logger.Debug("AutoSync: Skipping sync that needs emulator selection", "game", s.GameBase)
			hadSkipped = true
			a.addIssue(SyncIssue{Sync: *s, NeedsEmulator: true})
			continue
		}

		switch s.Action {
		case Upload:
			a.icon.SetText(icons.CloudUpload)
			time.Sleep(syncIconDelay)
			logger.Debug("AutoSync: Uploading", "game", s.GameBase)
		case Download:
			a.icon.SetText(icons.CloudDownload)
			time.Sleep(syncIconDelay)
			logger.Debug("AutoSync: Downloading", "game", s.GameBase)
		case Skip:
			continue
		}

		result := s.Execute(a.host)
		if !result.Success {
			logger.Error("AutoSync: Sync failed", "game", s.GameBase, "error", result.Error)
			hadError = true
			a.addIssue(SyncIssue{Sync: *s, ErrorMessage: result.Error})
		} else {
			logger.Debug("AutoSync: Sync successful", "game", s.GameBase, "action", result.Action)
		}
	}

	if hadError || hadSkipped {
		a.icon.SetText(icons.CloudAlert)
		a.hasIssues.Store(true)
		a.showButton.Store(true)
		logger.Debug("AutoSync: Completed with issues", "errors", hadError, "skipped", hadSkipped)
	} else {
		a.icon.SetText(icons.CloudCheck)
		a.showButton.Store(false)
		logger.Debug("AutoSync: Completed successfully")
	}
}
