package fiction

import (
	"fmt"
	"math"
	"strings"
)

// WordTracker provides utilities for tracking and managing word counts
type WordTracker struct {
	TargetWords    int                    `json:"target_words"`
	CurrentWords   int                    `json:"current_words"`
	ChapterTargets map[int]int            `json:"chapter_targets"`
	ChapterActuals map[int]int            `json:"chapter_actuals"`
	SceneTargets   map[string]int         `json:"scene_targets"`
	SceneActuals   map[string]int         `json:"scene_actuals"`
}

func NewWordTracker(targetWords int) *WordTracker {
	return &WordTracker{
		TargetWords:    targetWords,
		CurrentWords:   0,
		ChapterTargets: make(map[int]int),
		ChapterActuals: make(map[int]int),
		SceneTargets:   make(map[string]int),
		SceneActuals:   make(map[string]int),
	}
}

func (wt *WordTracker) SetChapterTarget(chapter, words int) {
	wt.ChapterTargets[chapter] = words
}

func (wt *WordTracker) SetSceneTarget(chapterNum, sceneNum, words int) {
	key := fmt.Sprintf("ch%d_sc%d", chapterNum, sceneNum)
	wt.SceneTargets[key] = words
}

func (wt *WordTracker) RecordScene(chapterNum, sceneNum int, content string) {
	key := fmt.Sprintf("ch%d_sc%d", chapterNum, sceneNum)
	words := CountWords(content)
	wt.SceneActuals[key] = words
	
	// Update chapter total
	wt.updateChapterActual(chapterNum)
	
	// Update total
	wt.updateTotal()
}

func (wt *WordTracker) updateChapterActual(chapterNum int) {
	total := 0
	for key, words := range wt.SceneActuals {
		if strings.HasPrefix(key, fmt.Sprintf("ch%d_", chapterNum)) {
			total += words
		}
	}
	wt.ChapterActuals[chapterNum] = total
}

func (wt *WordTracker) updateTotal() {
	total := 0
	for _, words := range wt.SceneActuals {
		total += words
	}
	wt.CurrentWords = total
}

func (wt *WordTracker) GetProgress() float64 {
	if wt.TargetWords == 0 {
		return 0
	}
	return float64(wt.CurrentWords) / float64(wt.TargetWords)
}

func (wt *WordTracker) GetChapterProgress(chapterNum int) (actual, target int, percentage float64) {
	actual = wt.ChapterActuals[chapterNum]
	target = wt.ChapterTargets[chapterNum]
	
	if target == 0 {
		return actual, target, 0
	}
	
	percentage = float64(actual) / float64(target)
	return
}

func (wt *WordTracker) GetSceneProgress(chapterNum, sceneNum int) (actual, target int, percentage float64) {
	key := fmt.Sprintf("ch%d_sc%d", chapterNum, sceneNum)
	actual = wt.SceneActuals[key]
	target = wt.SceneTargets[key]
	
	if target == 0 {
		return actual, target, 0
	}
	
	percentage = float64(actual) / float64(target)
	return
}

func (wt *WordTracker) NeedsAdjustment(chapterNum int, threshold float64) (bool, string) {
	actual, target, percentage := wt.GetChapterProgress(chapterNum)
	
	if percentage < (1.0 - threshold) {
		return true, fmt.Sprintf("Chapter %d is %d words short (%.1f%% of target)", 
			chapterNum, target-actual, percentage*100)
	}
	
	if percentage > (1.0 + threshold) {
		return true, fmt.Sprintf("Chapter %d is %d words over (%.1f%% of target)", 
			chapterNum, actual-target, percentage*100)
	}
	
	return false, ""
}

func (wt *WordTracker) GetSummary() string {
	return fmt.Sprintf(`Word Count Summary:
Target: %d words
Actual: %d words  
Progress: %.1f%%
Accuracy: %.1f%%

Chapters: %d/%d completed
Scenes: %d completed`,
		wt.TargetWords,
		wt.CurrentWords,
		wt.GetProgress()*100,
		math.Min(wt.GetProgress(), 1.0/wt.GetProgress())*100,
		len(wt.ChapterActuals),
		len(wt.ChapterTargets),
		len(wt.SceneActuals))
}

// CountWords counts words in text
func CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// EstimateReadingTime estimates reading time in minutes
func EstimateReadingTime(wordCount int) int {
	// Average reading speed: 200-250 words per minute
	return wordCount / 225
}

// CalculateOptimalChapterCount suggests chapter count for target length
func CalculateOptimalChapterCount(targetWords int) int {
	// Aim for 800-1200 words per chapter
	idealWordsPerChapter := 1000
	chapters := targetWords / idealWordsPerChapter
	
	if chapters < 5 {
		return 5 // Minimum for proper story structure
	}
	if chapters > 30 {
		return 30 // Maximum for readability
	}
	
	return chapters
}

// CalculateSceneDistribution suggests scenes per chapter
func CalculateSceneDistribution(wordsPerChapter int) int {
	// Aim for 250-400 words per scene
	idealWordsPerScene := 333
	scenes := wordsPerChapter / idealWordsPerScene
	
	if scenes < 2 {
		return 2 // Minimum for chapter structure
	}
	if scenes > 5 {
		return 5 // Maximum for readability
	}
	
	return scenes
}