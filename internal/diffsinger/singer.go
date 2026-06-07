/**************************************************************************
 * DiffScope Synthesis Platform                                           *
 * Copyright (C) 2026 Team OpenVPI                                        *
 *                                                                        *
 * This program is free software: you can redistribute it and/or modify   *
 * it under the terms of the GNU General Public License as published by   *
 * the Free Software Foundation, either version 3 of the License, or      *
 * (at your option) any later version.                                    *
 *                                                                        *
 * This program is distributed in the hope that it will be useful,        *
 * but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 * GNU General Public License for more details.                           *
 *                                                                        *
 * You should have received a copy of the GNU General Public License      *
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. *
 **************************************************************************/

package diffsinger

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"diffscope-synthesis-platform/internal/dsinfer"
	"diffscope-synthesis-platform/internal/synthrt"

	"github.com/diffscope/diffscope-package-manager/packagedatabase"
	"github.com/diffscope/diffscope-package-manager/packagedatabase/model"
	"github.com/diffscope/diffscope-package-manager/packageinfo"
)

var logger = slog.With("component", "diffsinger.singer")

const (
	diffSingerClass            = "diffsinger"
	packageDescriptionFileName = "desc.json"
)

type OnsetMode string

const (
	OnsetModeRule   OnsetMode = "rule"
	OnsetModeCustom OnsetMode = "custom"
)

type S2PMode string

const (
	S2PModeDirect S2PMode = "direct"
	S2PModeMap    S2PMode = "map"
	S2PModeDict   S2PMode = "dict"
	S2PModeCustom S2PMode = "custom"
)

type SingerIdentifier struct {
	PackageID string
	Version   packageinfo.PackageVersion
	SingerID  string
}

type SingerMetadata struct {
	PackageID        string
	Version          packageinfo.PackageVersion
	PackageDirectory string
	SingerConfigPath string

	Avatar     *packageinfo.MultilingualText
	Background *packageinfo.MultilingualText
	DemoAudio  []packageinfo.SingerDemoAudio
	Name       packageinfo.MultilingualText

	DefaultLanguage string
	G2PPackagesPath *string
	Languages       map[string]SingerLanguage
	Speakers        map[string]SingerSpeaker

	SynthRTSinger     *synthrt.Singer
	durationInference *dsinfer.DurationInference
}

type SingerLanguage struct {
	ID        string
	G2P       string
	OnsetFile string
	OnsetMode OnsetMode
	S2PFile   string
	S2PMode   S2PMode
}

type SingerSpeaker struct {
	ID   string
	Name packageinfo.MultilingualText
}

var (
	singerMetadataMu sync.RWMutex
	singerMetadata   = make(map[SingerIdentifier]SingerMetadata)
)

func GetSinger(id SingerIdentifier) (SingerMetadata, bool) {
	singerMetadataMu.RLock()
	defer singerMetadataMu.RUnlock()

	metadata, ok := singerMetadata[id]
	if !ok {
		return SingerMetadata{}, false
	}
	return cloneSingerMetadata(metadata), true
}

func ListSingers() []SingerMetadata {
	singerMetadataMu.RLock()
	defer singerMetadataMu.RUnlock()

	items := make([]SingerMetadata, 0, len(singerMetadata))
	for _, metadata := range singerMetadata {
		items = append(items, cloneSingerMetadata(metadata))
	}
	return items
}

func SingerMetadataMap() map[SingerIdentifier]SingerMetadata {
	singerMetadataMu.RLock()
	defer singerMetadataMu.RUnlock()

	items := make(map[SingerIdentifier]SingerMetadata, len(singerMetadata))
	for id, metadata := range singerMetadata {
		items[id] = cloneSingerMetadata(metadata)
	}
	return items
}

func RefreshSingerRegistry(packagesDir string) error {
	metadata, err := LoadSingerMetadata(packagesDir)
	if err != nil {
		return err
	}

	singerMetadataMu.Lock()
	defer singerMetadataMu.Unlock()
	singerMetadata = metadata
	return nil
}

func LoadSingerMetadata(packagesDir string) (map[SingerIdentifier]SingerMetadata, error) {
	if packagesDir == "" {
		return nil, fmt.Errorf("diffsinger: package directory path is required")
	}

	db, err := packagedatabase.Open(filepath.Join(packagesDir, "packages.db"))
	if err != nil {
		return nil, err
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	var rows []model.Singer
	if err := db.
		Where("class = ?", diffSingerClass).
		Order("package_id ASC, package_version ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	loadSingerTexts := func(packageID string, version string, singerID string) (singerTexts, error) {
		var rows []model.SingerMultilingualInfo
		if err := db.
			Where("package_id = ? AND package_version = ? AND singer_id = ?", packageID, version, singerID).
			Find(&rows).Error; err != nil {
			return singerTexts{}, err
		}

		var names, avatars, backgrounds packageinfo.MultilingualText
		for _, row := range rows {
			addOptionalMultilingualField(&names, row.Language, row.Name)
			addOptionalMultilingualField(&avatars, row.Language, row.Avatar)
			addOptionalMultilingualField(&backgrounds, row.Language, row.Background)
		}
		return singerTexts{
			Name:       normalizeMultilingualText(names),
			Avatar:     normalizeMultilingualText(avatars),
			Background: normalizeMultilingualText(backgrounds),
		}, nil
	}

	var loadDemoAudioText func(packageID string, version string, singerID string, demoIndex int) (packageinfo.SingerDemoAudio, error)

	loadDemoAudio := func(packageID string, version string, singerID string) ([]packageinfo.SingerDemoAudio, error) {
		var rows []model.SingerDemoAudio
		if err := db.
			Where("package_id = ? AND package_version = ? AND singer_id = ?", packageID, version, singerID).
			Order("`index` ASC").
			Find(&rows).Error; err != nil {
			return nil, err
		}

		demoAudio := make([]packageinfo.SingerDemoAudio, 0, len(rows))
		for _, row := range rows {
			item, err := loadDemoAudioText(row.PackageID, row.PackageVersion, row.SingerID, row.Index)
			if err != nil {
				return nil, err
			}
			demoAudio = append(demoAudio, item)
		}
		return demoAudio, nil
	}

	loadDemoAudioText = func(packageID string, version string, singerID string, demoIndex int) (packageinfo.SingerDemoAudio, error) {
		var rows []model.SingerDemoAudioMultilingualInfo
		if err := db.
			Where("package_id = ? AND package_version = ? AND singer_id = ? AND demo_index = ?", packageID, version, singerID, demoIndex).
			Find(&rows).Error; err != nil {
			return packageinfo.SingerDemoAudio{}, err
		}

		var names, paths packageinfo.MultilingualText
		for _, row := range rows {
			addOptionalMultilingualField(&names, row.Language, row.Name)
			addOptionalMultilingualField(&paths, row.Language, row.Audio)
		}
		name := normalizeMultilingualText(names)
		if name == nil {
			name = &packageinfo.MultilingualText{}
		}
		audioPath := normalizeMultilingualText(paths)
		if audioPath == nil {
			audioPath = &packageinfo.MultilingualText{}
		}
		return packageinfo.SingerDemoAudio{
			Name: *name,
			Path: *audioPath,
		}, nil
	}

	metadata := make(map[SingerIdentifier]SingerMetadata, len(rows))
	contributionPaths := make(map[string]map[string]string)
	logLoadError := func(packageID string, version string, singerID string, singerConfigPath string, err error) {
		logger.Error(
			"Failed to load singer metadata",
			slog.Any("error", err),
			slog.String("package_id", packageID),
			slog.String("version", version),
			slog.String("singer_id", singerID),
			slog.String("singer_config_path", singerConfigPath),
		)
	}

	for _, row := range rows {
		version, err := packageinfo.ParsePackageVersion(row.PackageVersion)
		if err != nil {
			logLoadError(row.PackageID, row.PackageVersion, row.ID, "", fmt.Errorf("parse package version %q: %w", row.PackageVersion, err))
			continue
		}

		id := SingerIdentifier{
			PackageID: row.PackageID,
			Version:   version,
			SingerID:  row.ID,
		}
		packageDir, err := filepath.Abs(installedPackageDir(packagesDir, row.PackageID, row.PackageVersion))
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, "", fmt.Errorf("resolve package directory: %w", err))
			continue
		}

		packageKey := row.PackageID + "\x00" + row.PackageVersion
		paths, ok := contributionPaths[packageKey]
		if !ok {
			paths, err = readSingerContributionPaths(packageDir)
			if err != nil {
				logLoadError(id.PackageID, id.Version.String(), id.SingerID, "", err)
				continue
			}
			contributionPaths[packageKey] = paths
		}

		singerConfigPath, ok := paths[row.ID]
		if !ok {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, "", fmt.Errorf("singer configuration path not found"))
			continue
		}

		texts, err := loadSingerTexts(row.PackageID, row.PackageVersion, row.ID)
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, singerConfigPath, err)
			continue
		}
		demoAudio, err := loadDemoAudio(row.PackageID, row.PackageVersion, row.ID)
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, singerConfigPath, err)
			continue
		}

		item, err := readSingerMetadata(row.PackageID, version, packageDir, singerConfigPath, row.ID, texts, demoAudio)
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, singerConfigPath, err)
			continue
		}

		srtPackage, err := synthrt.GetPackage(packageDir, id.PackageID, synthRTVersionNumber(id.Version))
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, singerConfigPath, fmt.Errorf("load SynthRT package: %w", err))
			continue
		}
		srtSinger, err := srtPackage.Singer(id.SingerID)
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, singerConfigPath, fmt.Errorf("load SynthRT singer: %w", err))
			continue
		}
		item.SynthRTSinger = srtSinger
		durationInference, err := dsinfer.GetDurationInference(srtSinger)
		if err != nil {
			logLoadError(id.PackageID, id.Version.String(), id.SingerID, singerConfigPath, fmt.Errorf("get duration inference: %w", err))
			continue
		}
		item.durationInference = durationInference

		metadata[id] = item
		logger.Info(
			"Loaded singer metadata",
			slog.String("package_id", id.PackageID),
			slog.String("version", id.Version.String()),
			slog.String("singer_id", id.SingerID),
		)
		logger.Debug("Loaded singer metadata detail", slog.Any("metadata", item))
	}
	return metadata, nil
}

type singerTexts struct {
	Name       *packageinfo.MultilingualText
	Avatar     *packageinfo.MultilingualText
	Background *packageinfo.MultilingualText
}

type packageContributionDescription struct {
	Contributes struct {
		Singers []string `json:"singers"`
	} `json:"contributes"`
}

type singerIDDescription struct {
	ID string `json:"id"`
}

func readSingerContributionPaths(packageDir string) (map[string]string, error) {
	descriptionPath := filepath.Join(packageDir, packageDescriptionFileName)
	data, err := os.ReadFile(descriptionPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", packageDescriptionFileName, err)
	}

	var description packageContributionDescription
	if err := json.Unmarshal(data, &description); err != nil {
		return nil, fmt.Errorf("parse %s: %w", packageDescriptionFileName, err)
	}

	paths := make(map[string]string, len(description.Contributes.Singers))
	for _, singerPath := range description.Contributes.Singers {
		cleaned := cleanPackageRelativePath(singerPath)
		if cleaned == "" {
			continue
		}
		absolutePath := filepath.Join(packageDir, filepath.FromSlash(cleaned))
		data, err := os.ReadFile(absolutePath)
		if err != nil {
			continue
		}
		var singer singerIDDescription
		if err := json.Unmarshal(data, &singer); err != nil {
			continue
		}
		if singer.ID != "" {
			paths[singer.ID] = absolutePath
		}
	}
	return paths, nil
}

func readSingerMetadata(
	packageID string,
	version packageinfo.PackageVersion,
	packageDir string,
	singerConfigPath string,
	singerID string,
	texts singerTexts,
	demoAudio []packageinfo.SingerDemoAudio,
) (SingerMetadata, error) {
	data, err := os.ReadFile(singerConfigPath)
	if err != nil {
		return SingerMetadata{}, fmt.Errorf("read singer configuration: %w", err)
	}

	var description packageinfo.SingerDescription
	if err := json.Unmarshal(data, &description); err != nil {
		return SingerMetadata{}, fmt.Errorf("parse singer configuration: %w", err)
	}

	name := texts.Name
	if name == nil {
		name = &packageinfo.MultilingualText{Default: singerID}
	}

	absolutizeMultilingualPath(texts.Avatar, packageDir)
	absolutizeMultilingualPath(texts.Background, packageDir)
	for index := range demoAudio {
		absolutizeMultilingualPath(&demoAudio[index].Path, packageDir)
	}

	metadata := SingerMetadata{
		PackageID:        packageID,
		Version:          version,
		PackageDirectory: packageDir,
		SingerConfigPath: singerConfigPath,
		Avatar:           texts.Avatar,
		Background:       texts.Background,
		DemoAudio:        demoAudio,
		Name:             *name,
		Languages:        make(map[string]SingerLanguage),
		Speakers:         make(map[string]SingerSpeaker),
	}

	if description.Configuration == nil {
		return metadata, nil
	}

	config, err := parseSingerConfiguration(*description.Configuration, filepath.Dir(singerConfigPath))
	if err != nil {
		return SingerMetadata{}, err
	}
	metadata.DefaultLanguage = config.DefaultLanguage
	metadata.G2PPackagesPath = config.G2PPackagesPath
	metadata.Languages = config.Languages
	metadata.Speakers = config.Speakers
	return metadata, nil
}

type rawSingerConfiguration struct {
	DefaultLanguage string        `json:"defaultLanguage"`
	G2PPackagesPath *string       `json:"g2pPackagesPath,omitempty"`
	Languages       []rawLanguage `json:"languages"`
	Speakers        []rawSpeaker  `json:"speakers"`
}

type parsedSingerConfiguration struct {
	DefaultLanguage string
	G2PPackagesPath *string
	Languages       map[string]SingerLanguage
	Speakers        map[string]SingerSpeaker
}

type rawLanguage struct {
	ID        string    `json:"id"`
	G2P       string    `json:"g2p"`
	OnsetFile string    `json:"onsetFile"`
	OnsetMode OnsetMode `json:"onsetMode"`
	S2PFile   string    `json:"s2pFile"`
	S2PMode   S2PMode   `json:"s2pMode"`
}

type rawSpeaker struct {
	ID   string                        `json:"id"`
	Name *packageinfo.MultilingualText `json:"name,omitempty"`
}

func parseSingerConfiguration(data json.RawMessage, baseDir string) (parsedSingerConfiguration, error) {
	var raw rawSingerConfiguration
	if err := json.Unmarshal(data, &raw); err != nil {
		return parsedSingerConfiguration{}, fmt.Errorf("parse singer configuration metadata: %w", err)
	}

	if raw.G2PPackagesPath != nil {
		path := absolutizePath(*raw.G2PPackagesPath, baseDir)
		raw.G2PPackagesPath = &path
	}

	languages := make(map[string]SingerLanguage, len(raw.Languages))
	for _, item := range raw.Languages {
		if item.ID == "" {
			return parsedSingerConfiguration{}, fmt.Errorf("language id cannot be empty")
		}
		if _, ok := languages[item.ID]; ok {
			return parsedSingerConfiguration{}, fmt.Errorf("duplicate language id %q", item.ID)
		}
		if !isValidOnsetMode(item.OnsetMode) {
			return parsedSingerConfiguration{}, fmt.Errorf("invalid onset mode %q for language %q", item.OnsetMode, item.ID)
		}
		if !isValidS2PMode(item.S2PMode) {
			return parsedSingerConfiguration{}, fmt.Errorf("invalid s2p mode %q for language %q", item.S2PMode, item.ID)
		}
		languages[item.ID] = SingerLanguage{
			ID:        item.ID,
			G2P:       item.G2P,
			OnsetFile: absolutizePath(item.OnsetFile, baseDir),
			OnsetMode: item.OnsetMode,
			S2PFile:   absolutizePath(item.S2PFile, baseDir),
			S2PMode:   item.S2PMode,
		}
	}

	speakers := make(map[string]SingerSpeaker, len(raw.Speakers))
	for _, item := range raw.Speakers {
		if item.ID == "" {
			return parsedSingerConfiguration{}, fmt.Errorf("speaker id cannot be empty")
		}
		if _, ok := speakers[item.ID]; ok {
			return parsedSingerConfiguration{}, fmt.Errorf("duplicate speaker id %q", item.ID)
		}
		name := item.Name
		if name == nil {
			name = &packageinfo.MultilingualText{Default: item.ID}
		}
		speakers[item.ID] = SingerSpeaker{
			ID:   item.ID,
			Name: cloneMultilingualTextValue(*name),
		}
	}

	return parsedSingerConfiguration{
		DefaultLanguage: raw.DefaultLanguage,
		G2PPackagesPath: raw.G2PPackagesPath,
		Languages:       languages,
		Speakers:        speakers,
	}, nil
}

func isValidOnsetMode(mode OnsetMode) bool {
	switch mode {
	case OnsetModeRule, OnsetModeCustom:
		return true
	default:
		return false
	}
}

func isValidS2PMode(mode S2PMode) bool {
	switch mode {
	case S2PModeDirect, S2PModeMap, S2PModeDict, S2PModeCustom:
		return true
	default:
		return false
	}
}

func installedPackageDir(packagesDir string, packageID string, version string) string {
	return filepath.Join(packagesDir, url.PathEscape(packageID)+"@"+version)
}

func synthRTVersionNumber(version packageinfo.PackageVersion) synthrt.VersionNumber {
	return synthrt.VersionNumber{
		Major: int(version.Major),
		Minor: int(version.Minor),
		Patch: int(version.Patch),
		Tweak: int(version.Build),
	}
}

func cleanPackageRelativePath(value string) string {
	normalized := strings.ReplaceAll(value, "\\", "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func addOptionalMultilingualField(text *packageinfo.MultilingualText, language string, value *string) {
	if value == nil {
		return
	}
	if language == "_" {
		text.Default = *value
		return
	}
	if text.Texts == nil {
		text.Texts = make(map[string]string)
	}
	text.Texts[language] = *value
}

func normalizeMultilingualText(text packageinfo.MultilingualText) *packageinfo.MultilingualText {
	if text.Default == "" && len(text.Texts) == 0 {
		return nil
	}
	if text.Texts == nil {
		text.Texts = make(map[string]string)
	}
	return &text
}

func absolutizeMultilingualPath(text *packageinfo.MultilingualText, baseDir string) {
	if text == nil {
		return
	}
	text.Default = absolutizePath(text.Default, baseDir)
	for language, value := range text.Texts {
		text.Texts[language] = absolutizePath(value, baseDir)
	}
}

func absolutizePath(value string, baseDir string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

func cloneSingerMetadata(metadata SingerMetadata) SingerMetadata {
	metadata.Avatar = cloneMultilingualText(metadata.Avatar)
	metadata.Background = cloneMultilingualText(metadata.Background)
	metadata.Name = cloneMultilingualTextValue(metadata.Name)
	metadata.DemoAudio = cloneDemoAudio(metadata.DemoAudio)
	if metadata.G2PPackagesPath != nil {
		value := *metadata.G2PPackagesPath
		metadata.G2PPackagesPath = &value
	}
	metadata.Languages = cloneLanguages(metadata.Languages)
	metadata.Speakers = cloneSpeakers(metadata.Speakers)
	return metadata
}

func cloneDemoAudio(items []packageinfo.SingerDemoAudio) []packageinfo.SingerDemoAudio {
	if items == nil {
		return nil
	}
	cloned := make([]packageinfo.SingerDemoAudio, len(items))
	for index, item := range items {
		cloned[index] = packageinfo.SingerDemoAudio{
			Name: cloneMultilingualTextValue(item.Name),
			Path: cloneMultilingualTextValue(item.Path),
		}
	}
	return cloned
}

func cloneLanguages(items map[string]SingerLanguage) map[string]SingerLanguage {
	if items == nil {
		return nil
	}
	cloned := make(map[string]SingerLanguage, len(items))
	for key, item := range items {
		cloned[key] = item
	}
	return cloned
}

func cloneSpeakers(items map[string]SingerSpeaker) map[string]SingerSpeaker {
	if items == nil {
		return nil
	}
	cloned := make(map[string]SingerSpeaker, len(items))
	for key, item := range items {
		item.Name = cloneMultilingualTextValue(item.Name)
		cloned[key] = item
	}
	return cloned
}

func cloneMultilingualText(text *packageinfo.MultilingualText) *packageinfo.MultilingualText {
	if text == nil {
		return nil
	}
	cloned := cloneMultilingualTextValue(*text)
	return &cloned
}

func cloneMultilingualTextValue(text packageinfo.MultilingualText) packageinfo.MultilingualText {
	if text.Texts == nil {
		return text
	}
	cloned := packageinfo.MultilingualText{
		Default: text.Default,
		Texts:   make(map[string]string, len(text.Texts)),
	}
	for key, value := range text.Texts {
		cloned.Texts[key] = value
	}
	return cloned
}
