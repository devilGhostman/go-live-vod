"use client";

import { useEffect, useState } from "react";
import Navbar from "./components/navbar";
import VideoPlayer from "./components/video-player";
import axios from "./axios/axios";
import VideoList from "./components/video-list";

export default function Home() {
  const [videos, setVideos] = useState();
  const [lives, setLives] = useState();
  const [selectedVideo, setselectedVideo] = useState();

  const handleVideo = (video) => {
    setselectedVideo(video);
  };

  const getAllVideos = async () => {
    await axios
      .get("/vod/")
      .then((response) => {
        setVideos(response.data.data);
      })
      .catch((error) => {
        console.log(error);
      });
  };

  const getAllLives = async () => {
    await axios
      .get("/livestream/")
      .then((response) => {
        setLives(response.data.data);
      })
      .catch((error) => {
        console.log(error);
      });
  };

  useEffect(() => {
    getAllVideos();
    getAllLives();
  }, []);

  console.log("videos", videos, lives);

  return (
    <div className="w-full h-[100%] bg-black">
      <Navbar />
      {selectedVideo && <VideoPlayer video={selectedVideo} />}
      {videos && <VideoList videos={videos} handleVideo={handleVideo} />}
      {lives && <VideoList videos={lives} handleVideo={handleVideo} live />}
      {!videos && !lives && (
        <div className="w-full h-screen flex justify-center items-center">
          <p className="text-white">No Videos Found</p>
        </div>
      )}
    </div>
  );
}
