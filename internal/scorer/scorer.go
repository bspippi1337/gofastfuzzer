package scorer

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type ScoreBoard struct {
	mu     sync.Mutex
	Scores map[string]float64 `json:"scores"`
}

func New() *ScoreBoard {
	return &ScoreBoard{Scores: map[string]float64{}}
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "scoreboard.json"
	}
	return filepath.Join(home, ".fastfuzzer", "scoreboard.json")
}

func (s *ScoreBoard) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, s)
}

func (s *ScoreBoard) Save(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (s *ScoreBoard) Update(action string, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.Scores[action]
	if success {
		v += 1.0
	} else {
		v -= 0.25
	}
	// mild decay to prevent runaway scores
	if v > 50 {
		v = 50
	}
	if v < -20 {
		v = -20
	}
	s.Scores[action] = v
}

// Pick chooses an index weighted by learned scores and a bit of randomness.
// baseWeight(i) should return a non-negative number.
func (s *ScoreBoard) Pick(keys []string, baseWeight func(i int) float64) int {
	s.mu.Lock()
	snap := make(map[string]float64, len(s.Scores))
	for k, v := range s.Scores {
		snap[k] = v
	}
	s.mu.Unlock()

	weights := make([]float64, len(keys))
	for i, k := range keys {
		bw := baseWeight(i)
		// map score to a multiplier in roughly [0.3, 3]
		score := snap[k]
		mult := 1.0 + (score / 10.0)
		if mult < 0.3 {
			mult = 0.3
		}
		if mult > 3 {
			mult = 3
		}
		w := bw * mult
		if w < 0 {
			w = 0
		}
		weights[i] = w
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		return 0
	}
	t := r.Float64() * sum
	acc := 0.0
	for i, w := range weights {
		acc += w
		if acc >= t {
			return i
		}
	}
	return len(keys) - 1
}

func (s *ScoreBoard) Top(n int) []struct {
	Action string
	Score  float64
} {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := make([]struct {
		Action string
		Score  float64
	}, 0, len(s.Scores))
	for k, v := range s.Scores {
		t = append(t, struct {
			Action string
			Score  float64
		}{k, v})
	}
	sort.Slice(t, func(i, j int) bool { return t[i].Score > t[j].Score })
	if n > 0 && len(t) > n {
		return t[:n]
	}
	return t
}
