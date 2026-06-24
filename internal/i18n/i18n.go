package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Locale string

const (
	Auto Locale = "auto"
	En   Locale = "en"
	ZhCN Locale = "zh-CN"
)

type Key string

type Translator struct {
	locale Locale
}

const (
	AppTitle Key = "app.title"

	MenuSettings          Key = "menu.settings"
	MenuPasteNow          Key = "menu.paste_now"
	MenuCheckUpdate       Key = "menu.check_update"
	MenuDownloadPortable  Key = "menu.download_portable"
	MenuDownloadInstaller Key = "menu.download_installer"
	MenuRepository        Key = "menu.repository"
	MenuQuit              Key = "menu.quit"

	LabelHotkey        Key = "label.hotkey"
	LabelLanguage      Key = "label.language"
	LabelStartDelay    Key = "label.start_delay"
	LabelInterKeyDelay Key = "label.inter_key_delay"
	LabelBatchSize     Key = "label.batch_size"
	LabelBatchPause    Key = "label.batch_pause"

	ButtonSave       Key = "button.save"
	ButtonPaste      Key = "button.paste"
	ButtonUpdate     Key = "button.update"
	ButtonRepository Key = "button.repository"

	StatusReady                     Key = "status.ready"
	StatusHotkeyUnavailable         Key = "status.hotkey_unavailable"
	StatusSaved                     Key = "status.saved"
	StatusSavedHotkeyRegisterFailed Key = "status.saved_hotkey_register_failed"
	StatusClipboardEmpty            Key = "status.clipboard_empty"
	StatusPasting                   Key = "status.pasting"
	StatusPasteFinished             Key = "status.paste_finished"
	StatusPasteFinishedHotkeyFailed Key = "status.paste_finished_hotkey_failed"
	StatusPasteFailed               Key = "status.paste_failed"
	StatusCheckingUpdate            Key = "status.checking_update"
	StatusUpdateCheckFailed         Key = "status.update_check_failed"
	StatusUpdateAvailable           Key = "status.update_available"
	StatusCurrentVersionUpToDate    Key = "status.current_version_up_to_date"
	StatusDownloading               Key = "status.downloading"
	StatusDownloadFailed            Key = "status.download_failed"
	StatusDownloaded                Key = "status.downloaded"
	StatusInvalidRepositoryURL      Key = "status.invalid_repository_url"
	StatusOpenRepositoryFailed      Key = "status.open_repository_failed"

	DialogUpdateTitle Key = "dialog.update.title"

	DiagConfig Key = "diag.config"
	DiagDriver Key = "diag.driver"

	CLIUsage                     Key = "cli.usage"
	CLIUnknownCommand            Key = "cli.unknown_command"
	CLIConfigLoadFailed          Key = "cli.config_load_failed"
	CLIPasteInputFailed          Key = "cli.paste_input_failed"
	CLIDryRunFailed              Key = "cli.dry_run_failed"
	CLIPasteFailed               Key = "cli.paste_failed"
	CLIDoctorHeader              Key = "cli.doctor_header"
	CLIDoctorConfig              Key = "cli.doctor_config"
	CLIDoctorHotkey              Key = "cli.doctor_hotkey"
	CLIDoctorPaste               Key = "cli.doctor_paste"
	CLIDoctorDriver              Key = "cli.doctor_driver"
	CLIDoctorClipboardOK         Key = "cli.doctor_clipboard_ok"
	CLIDoctorClipboardWarning    Key = "cli.doctor_clipboard_warning"
	CLIConfigRequiresCommand     Key = "cli.config_requires_command"
	CLIConfigSetUsage            Key = "cli.config_set_usage"
	CLIConfigGetFailed           Key = "cli.config_get_failed"
	CLIConfigSetFailed           Key = "cli.config_set_failed"
	CLIConfigSaveFailed          Key = "cli.config_save_failed"
	CLIConfigResetFailed         Key = "cli.config_reset_failed"
	CLIConfigSaved               Key = "cli.config_saved"
	CLIConfigReset               Key = "cli.config_reset"
	CLIUnknownConfigCommand      Key = "cli.unknown_config_command"
	CLIUpdateRequiresCommand     Key = "cli.update_requires_command"
	CLIUpdateCheckFailed         Key = "cli.update_check_failed"
	CLIUpdateAvailable           Key = "cli.update_available"
	CLIUpdateUpToDate            Key = "cli.update_up_to_date"
	CLIUnknownUpdateCommand      Key = "cli.unknown_update_command"
	CLIUpdateSelectAssetFailed   Key = "cli.update_select_asset_failed"
	CLIUpdateDownloadAssetFailed Key = "cli.update_download_asset_failed"
	CLIUpdateDownloaded          Key = "cli.update_downloaded"
)

var translations = map[Locale]map[Key]string{
	En: {
		AppTitle: "Paste Tool",

		MenuSettings:          "Settings",
		MenuPasteNow:          "Paste Now",
		MenuCheckUpdate:       "Check Update",
		MenuDownloadPortable:  "Download Portable",
		MenuDownloadInstaller: "Download Installer",
		MenuRepository:        "Repository",
		MenuQuit:              "Quit",

		LabelHotkey:        "Hotkey",
		LabelLanguage:      "Language",
		LabelStartDelay:    "Start delay (ms)",
		LabelInterKeyDelay: "Inter-key delay (ms)",
		LabelBatchSize:     "Batch size",
		LabelBatchPause:    "Batch pause (ms)",

		ButtonSave:       "Save",
		ButtonPaste:      "Paste",
		ButtonUpdate:     "Update",
		ButtonRepository: "Repository",

		StatusReady:                     "Ready: %s",
		StatusHotkeyUnavailable:         "Hotkey unavailable: %s",
		StatusSaved:                     "Saved: %s",
		StatusSavedHotkeyRegisterFailed: "Saved, but hotkey registration failed: %s",
		StatusClipboardEmpty:            "Clipboard is empty",
		StatusPasting:                   "Pasting...",
		StatusPasteFinished:             "Paste finished",
		StatusPasteFinishedHotkeyFailed: "Paste finished; hotkey registration failed: %s",
		StatusPasteFailed:               "Paste failed: %s",
		StatusCheckingUpdate:            "Checking update...",
		StatusUpdateCheckFailed:         "Update check failed: %s",
		StatusUpdateAvailable:           "Update available: %s -> %s",
		StatusCurrentVersionUpToDate:    "Current version %s is up to date",
		StatusDownloading:               "Downloading %s...",
		StatusDownloadFailed:            "Download failed: %s",
		StatusDownloaded:                "Downloaded: %s",
		StatusInvalidRepositoryURL:      "Invalid repository URL: %s",
		StatusOpenRepositoryFailed:      "Open repository failed: %s",

		DialogUpdateTitle: "Update",

		DiagConfig: "Config",
		DiagDriver: "Driver",

		CLIUsage: `Usage:
  paste-tool                         Launch tray GUI
  paste-tool gui                     Launch tray GUI
  paste-tool paste [flags]           Type text into the focused target
  paste-tool doctor                  Print platform and config diagnostics
  paste-tool config get [key]        Print config
  paste-tool config set <key> <val>  Update config
  paste-tool update check            Check GitHub latest release
  paste-tool update download [kind]  Download latest portable or installer
  paste-tool version                 Print version`,
		CLIUnknownCommand:            "unknown command %q",
		CLIConfigLoadFailed:          "config: %v",
		CLIPasteInputFailed:          "paste input: %v",
		CLIDryRunFailed:              "dry-run: %v",
		CLIPasteFailed:               "paste: %v",
		CLIDoctorHeader:              "%s %s",
		CLIDoctorConfig:              "config: %s",
		CLIDoctorHotkey:              "hotkey: %s",
		CLIDoctorPaste:               "paste: start_delay=%dms inter_key=%dms batch_size=%d batch_pause=%dms",
		CLIDoctorDriver:              "driver: %s",
		CLIDoctorClipboardOK:         "[ok] clipboard: text clipboard backend initialized",
		CLIDoctorClipboardWarning:    "[warning] clipboard: %v",
		CLIConfigRequiresCommand:     "config requires get, set, path, or reset",
		CLIConfigSetUsage:            "usage: paste-tool config set <key> <value>",
		CLIConfigGetFailed:           "config get: %v",
		CLIConfigSetFailed:           "config set: %v",
		CLIConfigSaveFailed:          "config save: %v",
		CLIConfigResetFailed:         "config reset: %v",
		CLIConfigSaved:               "saved %s",
		CLIConfigReset:               "reset %s",
		CLIUnknownConfigCommand:      "unknown config command %q",
		CLIUpdateRequiresCommand:     "update requires check or download",
		CLIUpdateCheckFailed:         "update check: %v",
		CLIUpdateAvailable:           "update available: %s -> %s\n%s",
		CLIUpdateUpToDate:            "current version %s is up to date against latest %s",
		CLIUnknownUpdateCommand:      "unknown update command %q",
		CLIUpdateSelectAssetFailed:   "select asset: %v",
		CLIUpdateDownloadAssetFailed: "download asset: %v",
		CLIUpdateDownloaded:          "downloaded %s",
	},
	ZhCN: {
		AppTitle: "粘贴工具",

		MenuSettings:          "设置",
		MenuPasteNow:          "立即粘贴",
		MenuCheckUpdate:       "检查更新",
		MenuDownloadPortable:  "下载便携版",
		MenuDownloadInstaller: "下载安装包",
		MenuRepository:        "仓库主页",
		MenuQuit:              "退出",

		LabelHotkey:        "热键",
		LabelLanguage:      "语言",
		LabelStartDelay:    "启动延迟（毫秒）",
		LabelInterKeyDelay: "字符间隔（毫秒）",
		LabelBatchSize:     "批量大小",
		LabelBatchPause:    "批间暂停（毫秒）",

		ButtonSave:       "保存",
		ButtonPaste:      "粘贴",
		ButtonUpdate:     "更新",
		ButtonRepository: "仓库",

		StatusReady:                     "就绪：%s",
		StatusHotkeyUnavailable:         "热键不可用：%s",
		StatusSaved:                     "已保存：%s",
		StatusSavedHotkeyRegisterFailed: "已保存，但热键注册失败：%s",
		StatusClipboardEmpty:            "剪贴板为空",
		StatusPasting:                   "正在粘贴...",
		StatusPasteFinished:             "粘贴完成",
		StatusPasteFinishedHotkeyFailed: "粘贴完成，但热键重新注册失败：%s",
		StatusPasteFailed:               "粘贴失败：%s",
		StatusCheckingUpdate:            "正在检查更新...",
		StatusUpdateCheckFailed:         "检查更新失败：%s",
		StatusUpdateAvailable:           "发现新版本：%s -> %s",
		StatusCurrentVersionUpToDate:    "当前版本 %s 已是最新",
		StatusDownloading:               "正在下载 %s...",
		StatusDownloadFailed:            "下载失败：%s",
		StatusDownloaded:                "已下载：%s",
		StatusInvalidRepositoryURL:      "仓库地址无效：%s",
		StatusOpenRepositoryFailed:      "打开仓库失败：%s",

		DialogUpdateTitle: "更新",

		DiagConfig: "配置",
		DiagDriver: "驱动",

		CLIUsage: `用法：
  paste-tool                         启动托盘 GUI
  paste-tool gui                     启动托盘 GUI
  paste-tool paste [flags]           向当前焦点目标输入文本
  paste-tool doctor                  输出平台和配置诊断
  paste-tool config get [key]        输出配置
  paste-tool config set <key> <val>  修改配置
  paste-tool update check            检查 GitHub 最新版本
  paste-tool update download [kind]  下载最新便携版或安装包
  paste-tool version                 输出版本`,
		CLIUnknownCommand:            "未知命令 %q",
		CLIConfigLoadFailed:          "配置错误：%v",
		CLIPasteInputFailed:          "粘贴输入错误：%v",
		CLIDryRunFailed:              "dry-run 失败：%v",
		CLIPasteFailed:               "粘贴失败：%v",
		CLIDoctorHeader:              "%s %s",
		CLIDoctorConfig:              "配置：%s",
		CLIDoctorHotkey:              "热键：%s",
		CLIDoctorPaste:               "粘贴：启动延迟=%dms 字符间隔=%dms 批量大小=%d 批间暂停=%dms",
		CLIDoctorDriver:              "驱动：%s",
		CLIDoctorClipboardOK:         "[ok] 剪贴板：文本剪贴板后端已初始化",
		CLIDoctorClipboardWarning:    "[warning] 剪贴板：%v",
		CLIConfigRequiresCommand:     "config 需要 get、set、path 或 reset",
		CLIConfigSetUsage:            "用法：paste-tool config set <key> <value>",
		CLIConfigGetFailed:           "读取配置失败：%v",
		CLIConfigSetFailed:           "设置配置失败：%v",
		CLIConfigSaveFailed:          "保存配置失败：%v",
		CLIConfigResetFailed:         "重置配置失败：%v",
		CLIConfigSaved:               "已保存 %s",
		CLIConfigReset:               "已重置 %s",
		CLIUnknownConfigCommand:      "未知 config 命令 %q",
		CLIUpdateRequiresCommand:     "update 需要 check 或 download",
		CLIUpdateCheckFailed:         "检查更新失败：%v",
		CLIUpdateAvailable:           "发现新版本：%s -> %s\n%s",
		CLIUpdateUpToDate:            "当前版本 %s 已是最新；GitHub 最新版本为 %s",
		CLIUnknownUpdateCommand:      "未知 update 命令 %q",
		CLIUpdateSelectAssetFailed:   "选择资产失败：%v",
		CLIUpdateDownloadAssetFailed: "下载资产失败：%v",
		CLIUpdateDownloaded:          "已下载 %s",
	},
}

func New(locale string) Translator {
	return Translator{locale: Resolve(locale)}
}

func Resolve(value string) Locale {
	normalized := Normalize(value)
	if normalized == Auto {
		return Detect()
	}
	return normalized
}

func Normalize(value string) Locale {
	value = strings.TrimSpace(value)
	if value == "" {
		return Auto
	}
	lower := strings.ToLower(strings.ReplaceAll(value, "_", "-"))
	switch lower {
	case "auto", "system":
		return Auto
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn", "cn":
		return ZhCN
	case "en", "en-us", "en-gb":
		return En
	default:
		return Auto
	}
}

func Detect() Locale {
	if locale := normalizeEnvLocale(); locale != Auto {
		return locale
	}
	if locale := systemLocale(); locale != Auto {
		return locale
	}
	return En
}

func (t Translator) Locale() Locale {
	if t.locale == "" {
		return En
	}
	return t.locale
}

func (t Translator) T(key Key, args ...any) string {
	locale := t.Locale()
	format, ok := translations[locale][key]
	if !ok {
		format, ok = translations[En][key]
	}
	if !ok {
		format = string(key)
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func Supported() []Locale {
	return []Locale{Auto, ZhCN, En}
}

func normalizeEnvLocale() Locale {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale := Normalize(os.Getenv(name)); locale != Auto {
			return locale
		}
	}
	return Auto
}
