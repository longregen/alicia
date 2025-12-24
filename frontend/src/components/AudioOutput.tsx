import { useState, useEffect, useRef } from 'react';
import { Room, Track, RemoteTrack, RemoteParticipant, RoomEvent } from 'livekit-client';

interface AudioOutputProps {
  room: Room | null;
}

export function AudioOutput({ room }: AudioOutputProps) {
  const [isMuted, setIsMuted] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const audioElementRef = useRef<HTMLAudioElement>(null);
  const currentTrackRef = useRef<RemoteTrack | null>(null);

  useEffect(() => {
    if (!room) {
      setIsPlaying(false);
      return;
    }

    const handleTrackSubscribed = (
      track: RemoteTrack,
      _publication: any,
      _participant: RemoteParticipant
    ) => {
      if (track.kind === Track.Kind.Audio) {
        currentTrackRef.current = track;

        // Attach audio track to audio element
        if (audioElementRef.current) {
          track.attach(audioElementRef.current);
          audioElementRef.current.muted = isMuted;
          setIsPlaying(true);
        }
      }
    };

    const handleTrackUnsubscribed = (
      track: RemoteTrack,
      _publication: any,
      _participant: RemoteParticipant
    ) => {
      if (track.kind === Track.Kind.Audio) {
        // Detach audio track
        if (audioElementRef.current) {
          track.detach(audioElementRef.current);
        }

        if (currentTrackRef.current === track) {
          currentTrackRef.current = null;
          setIsPlaying(false);
        }
      }
    };

    // Check for existing audio tracks
    room.remoteParticipants.forEach(participant => {
      participant.audioTrackPublications.forEach(publication => {
        if (publication.track && publication.isSubscribed) {
          handleTrackSubscribed(publication.track as RemoteTrack, publication, participant);
        }
      });
    });

    // Listen for new tracks
    room.on(RoomEvent.TrackSubscribed, handleTrackSubscribed);
    room.on(RoomEvent.TrackUnsubscribed, handleTrackUnsubscribed);

    return () => {
      room.off(RoomEvent.TrackSubscribed, handleTrackSubscribed);
      room.off(RoomEvent.TrackUnsubscribed, handleTrackUnsubscribed);

      // Detach current track
      if (currentTrackRef.current && audioElementRef.current) {
        currentTrackRef.current.detach(audioElementRef.current);
      }
    };
  }, [room, isMuted]);

  const toggleMute = () => {
    const newMuted = !isMuted;
    setIsMuted(newMuted);

    if (audioElementRef.current) {
      audioElementRef.current.muted = newMuted;
    }
  };

  if (!room) {
    return null;
  }

  return (
    <div className="audio-output">
      <audio ref={audioElementRef} autoPlay playsInline />

      <div className="audio-controls">
        {isPlaying && (
          <>
            <div className="playing-indicator" style={{ fontSize: '12px', marginRight: '8px' }}>
              🔊 Assistant speaking
            </div>
            <button
              className="mute-btn"
              onClick={toggleMute}
              title={isMuted ? 'Unmute' : 'Mute'}
              style={{
                padding: '4px 8px',
                fontSize: '12px',
                cursor: 'pointer',
              }}
            >
              {isMuted ? '🔇 Muted' : '🔊 Unmute'}
            </button>
          </>
        )}
      </div>
    </div>
  );
}
