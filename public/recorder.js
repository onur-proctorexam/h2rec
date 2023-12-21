export function newMediaRecorder(mediaStream, options = null) {
  return new MediaRecorder(mediaStream, getOptions(mediaStream, options));
}

function getOptions(mediaStream, options) {
  return {
    mimeType: getMimeType(mediaStream),
    ...options,
  };
}

function getMimeType(mediaStream) {
  let preferedTypes;
  let hv = hasVideo(mediaStream);
  let ha = hasAudio(mediaStream);
  if (hv && ha) {
    preferedTypes = ['video/webm;codecs="vp9,opus"', 'video/webm;codecs="vp8,opus"', "video/webm"];
  } else if (hv) {
    preferedTypes = ['video/webm;codecs="vp9"', 'video/webm;codecs="vp8"', "video/webm"];
  } else if (ha) {
    preferedTypes = ['audio/webm;codecs="opus"', "audio/webm"];
  }
  const mimeType = preferedTypes.find((c) => MediaRecorder.isTypeSupported(c));
  if (!mimeType) {
    throw new Error("unsupported mime type");
  }
  return mimeType;
}

function hasVideo(mediaStream) {
  return mediaStream.getVideoTracks().length > 0;
}

function hasAudio(mediaStream) {
  return mediaStream.getAudioTracks().length > 0;
}
