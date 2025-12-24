import { useState, useEffect, useRef } from 'react';

interface AudioInputProps {
  onTrackReady: (track: MediaStreamTrack) => void;
  onTrackStop: () => void;
  disabled: boolean;
}

export function AudioInput({ onTrackReady, onTrackStop, disabled }: AudioInputProps) {
  const [isRecording, setIsRecording] = useState(false);
  const [audioLevel, setAudioLevel] = useState(0);
  const [permissionStatus, setPermissionStatus] = useState<'granted' | 'denied' | 'prompt'>('prompt');
  const [error, setError] = useState<string | null>(null);

  const streamRef = useRef<MediaStream | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const animationFrameRef = useRef<number | null>(null);

  // Request microphone permission
  const requestPermission = async () => {
    try {
      setError(null);
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        }
      });

      streamRef.current = stream;
      setPermissionStatus('granted');
      return stream;
    } catch (err) {
      console.error('Microphone permission denied:', err);
      setError('Microphone access denied');
      setPermissionStatus('denied');
      return null;
    }
  };

  // Set up audio level visualization
  const setupAudioAnalyzer = (stream: MediaStream) => {
    const audioContext = new AudioContext();
    const analyser = audioContext.createAnalyser();
    const source = audioContext.createMediaStreamSource(stream);

    analyser.fftSize = 256;
    source.connect(analyser);

    audioContextRef.current = audioContext;
    analyserRef.current = analyser;

    const dataArray = new Uint8Array(analyser.frequencyBinCount);

    const updateLevel = () => {
      if (!analyserRef.current) return;

      analyserRef.current.getByteFrequencyData(dataArray);
      const average = dataArray.reduce((a, b) => a + b) / dataArray.length;
      setAudioLevel(average / 255); // Normalize to 0-1

      animationFrameRef.current = requestAnimationFrame(updateLevel);
    };

    updateLevel();
  };

  // Stop audio analyzer
  const stopAudioAnalyzer = () => {
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
    if (audioContextRef.current) {
      audioContextRef.current.close();
      audioContextRef.current = null;
    }
    analyserRef.current = null;
    setAudioLevel(0);
  };

  // Toggle recording
  const toggleRecording = async () => {
    if (isRecording) {
      // Stop recording
      if (streamRef.current) {
        streamRef.current.getTracks().forEach(track => track.stop());
        streamRef.current = null;
      }
      stopAudioAnalyzer();
      setIsRecording(false);
      onTrackStop();
    } else {
      // Start recording
      let stream = streamRef.current;
      if (!stream) {
        stream = await requestPermission();
        if (!stream) return;
      }

      const audioTrack = stream.getAudioTracks()[0];
      if (audioTrack) {
        setupAudioAnalyzer(stream);
        setIsRecording(true);
        onTrackReady(audioTrack);
      }
    }
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach(track => track.stop());
      }
      stopAudioAnalyzer();
    };
  }, []);

  return (
    <div className="audio-input">
      <button
        className={`record-btn ${isRecording ? 'recording' : ''}`}
        onClick={toggleRecording}
        disabled={disabled}
        title={isRecording ? 'Stop recording' : 'Start recording'}
      >
        <div className="mic-icon">
          {isRecording ? '⏸' : '🎤'}
        </div>
      </button>

      {isRecording && (
        <div className="audio-level-container">
          <div
            className="audio-level-bar"
            style={{
              width: '100px',
              height: '4px',
              background: '#ddd',
              borderRadius: '2px',
              overflow: 'hidden',
            }}
          >
            <div
              className="audio-level-fill"
              style={{
                width: `${audioLevel * 100}%`,
                height: '100%',
                background: '#4CAF50',
                transition: 'width 0.1s ease',
              }}
            />
          </div>
        </div>
      )}

      {error && (
        <div className="audio-error" style={{ color: 'red', fontSize: '12px', marginTop: '4px' }}>
          {error}
        </div>
      )}

      {permissionStatus === 'denied' && (
        <div className="permission-denied" style={{ fontSize: '12px', marginTop: '4px' }}>
          Please enable microphone access in your browser settings
        </div>
      )}
    </div>
  );
}
