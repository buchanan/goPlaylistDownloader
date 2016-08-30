package plexAPI

type Video struct {
	RatingKey string `xml:"ratingKey,attr"`
	Key string `xml:"key,attr"`
	Guid string `xml:"guid,attr"`
	Studio string `xml:"studio,attr"`
	Class string `xml:"type,attr"`
	Title string `xml:"title,attr"`
	ContentRating string `xml:"contentRating,attr"`
	Summary string `xml:"summary,attr"`
	Rating string `xml:"rating,attr"`
	Year string `xml:"year,attr"`
	Tagline string `xml:"tagline,attr"`
	Thumb string `xml:"thumb,attr"`
	ThumbPath string
	Art string `xml:"art,attr"`
	Duration string `xml:"duration,attr"`
	OriginallyAvailableAt string `xml:"originallyAvailableAt,attr"`
	AddedAt string `xml:"addedAt,attr"`
	UpdatedAt string `xml:"updatedAt,attr"`
	ChapterSource string `xml:"chapterSource,attr"`
	PrimaryExtraKey string `xml:"primaryExtraKey,attr"`
	Media []Media `xml:"Media"`
	Genres []Genre `xml:"Genre"`
	PlaylistItemID string `xml:"playlistItemID,attr"`
}

type Media struct{
	VideoResolution string `xml:"videoResolution,attr"`
	Id string `xml:"id,attr"`
	Duration string `xml:"duration,attr"`
	Bitrate string `xml:"bitrate,attr"`
	Width string `xml:"width,attr"`
	Height string `xml:"height,attr"`
	AspectRatio string `xml:"aspectRatio,attr"`
	AudioChannels string `xml:"audioChannels,attr"`
	AudioCodec string `xml:"audioCodec,attr"`
	VideoCodec string `xml:"videoCodec,attr"`
	Container string `xml:"container,attr"`
	VideoFrameRate string `xml:"videoFrameRate,attr"`
	OptimizedForStreaming string `xml:"optimizedForStreaming,attr"`
	Has64bitOffsets string `xml:"has64bitOffsets,attr"`
	Part Part `xml:"Part"`
}

type Part struct{
	Id string `xml:"id,attr"`
	Key string `xml:"key,attr"`
	Duration string `xml:"duration,attr"`
	File string `xml:"file,attr"`
	Size string `xml:"size,attr"`
	Container string `xml:"container,attr"`
	Has64bitOffsets string `xml:"has64bitOffsets,attr"`
	OptimizedForStreaming string `xml:"optimizedForStreaming,attr"`
	Streams []Stream `xml:"Stream"`
}

type Stream struct{
	Id string `xml:"id,attr"`
	StreamType string `xml:"streamType,attr"`
	Codec string `xml:"codec,attr"`
	Index string `xml:"index,attr"`
	Channels string `xml:"channels,attr"`
	BitRate string `xml:"bitrate,attr"`
	BitDepth string `xml:"bitDepth,attr"`
	Language string `xml:"language,attr"`
	LanguageCode string `xml:"languageCode,attr"`
	BitRateMode string `xml:"bitrateMode"`
	Cabac string `xml:"cabac,attr"`
	ChromaSubsampling string `xml:"chromaSubsampling,attr"`
	CodecID string `xml:"codecID,attr"`
	ColorSpace string `xml:"colorSpace,attr"`
	Duration string `xml:"duration,attr"`
	FrameRate string `xml:"frameRate,attr"`
	FrameRateMode string `xml:"frameRateMode,attr"`
	HasScalingMatrix string `xml:"hasScalingMatrix,attr"`
	Height string `xml:"height,attr"`
	Level string `xml:"level,attr"`
	Profile string `xml:"profile,attr"`
	RefFrames string `xml:"refFrames,attr"`
	SamplingRate string `xml:"samplingRate,attr"`
	ScanType string `xml:"scanType,attr"`
	StreamIdentifier string `xml:"streamIdentifier,attr"`
	Title string `xml:"title,attr"`
	Width string `xml:"width,attr"`
}

type Genre struct{
	Id string `xml:"id,attr"`
	Tag string `xml:"tag,attr"`
	Count string `xml:"count,attr"`
}