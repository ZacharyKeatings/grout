package constants

var LibretroCoreToBIOS = mustLoadJSONMap[string, CoreBIOS]("bios/core_requirements.json")
var PlatformToLibretroCores = mustLoadJSONMap[string, []string]("bios/platform_cores.json")
