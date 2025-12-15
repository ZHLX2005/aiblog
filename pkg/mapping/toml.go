package mapping

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// AudioMapping represents the mapping configuration for audio files
type AudioMapping struct {
	AudioFile string `toml:"audio_file"` // Absolute path to audio file
	ImageFile string `toml:"image_file"` // Absolute path to image file (to be filled by user)
	Speaker   string `toml:"speaker"`    // Speaker identifier (S1/S2)
	Content   string `toml:"content"`    // Brief content description for reference
}

// MappingFile represents the complete mapping file structure
type MappingFile struct {
	Title       string         `toml:"title"`        // Blog post title
	Description string         `toml:"description"`  // Blog description
	OutputPath  string         `toml:"output_path"`  // Suggested output path
	AudioCount  int            `toml:"audio_count"`  // Total number of audio files
	AudioFiles  []AudioMapping `toml:"audio_files"`  // List of audio mappings
}

// NewMappingFile creates a new mapping file structure
func NewMappingFile(title, description string) *MappingFile {
	return &MappingFile{
		Title:       title,
		Description: description,
		OutputPath:  "./composed_video.mp4",
		AudioFiles:  make([]AudioMapping, 0),
	}
}

// AddAudio adds an audio file to the mapping
func (m *MappingFile) AddAudio(audioPath, speaker, content string) {
	absPath, _ := filepath.Abs(audioPath)
	mapping := AudioMapping{
		AudioFile: absPath,
		ImageFile: "", // To be filled by user
		Speaker:   speaker,
		Content:   content,
	}
	m.AudioFiles = append(m.AudioFiles, mapping)
	m.AudioCount = len(m.AudioFiles)
}

// SaveToFile saves the mapping configuration to a TOML file
func (m *MappingFile) SaveToFile(filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write TOML
	encoder := toml.NewEncoder(file)
	encoder.Indent = "  "
	if err := encoder.Encode(m); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
	}

	return nil
}

// LoadMappingFile loads a mapping configuration from a TOML file
func LoadMappingFile(filePath string) (*MappingFile, error) {
	var mapping MappingFile

	// Read and decode TOML
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if err := toml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}

	return &mapping, nil
}

// Validate validates the mapping configuration
func (m *MappingFile) Validate() error {
	if len(m.AudioFiles) == 0 {
		return fmt.Errorf("no audio files in mapping")
	}

	for i, audio := range m.AudioFiles {
		if audio.AudioFile == "" {
			return fmt.Errorf("audio file %d: empty audio path", i+1)
		}
		if audio.ImageFile == "" {
			return fmt.Errorf("audio file %d: empty image path (please fill in the image_file field)", i+1)
		}
	}

	return nil
}