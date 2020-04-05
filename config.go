package picster

// Configuration holds the possible parameter values
type Configuration struct {
	folderNamePattern string
	fileNamePattern   string
	pictureFolder     string
	videoFolder       string
}

const defaultFolderNamePattern = "2006-01"
const defaultFileNamePattern = "20060102_150405"
const defaultPictureFolder = "fotos"
const defaultVideoFolder = "videos"

func getDefaultConfiguration() (config Configuration) {
	config = Configuration{
		defaultFolderNamePattern,
		defaultFileNamePattern,
		defaultPictureFolder,
		defaultVideoFolder}

	return
}

// GetConfiguration returns Configuration with current values
func GetConfiguration() (config Configuration) {

	config = getDefaultConfiguration()
	return
}
