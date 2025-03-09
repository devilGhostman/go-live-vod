import Link from "next/link";
import React from "react";

const VideoList = ({ videos, handleVideo, live }) => {
  // console.log("videos", videos);
  const handleOnClick = (video) => {
    if (video.status === "completed" || video.status === "live") {
      handleVideo(video);
    }
  };
  return (
    <div className="w-full h-auto my-10 px-5">
      <h2 className="my-4 text-[18px] font-bold">
        {!live ? "Transcoded Videos" : "Lives"}
      </h2>
      {videos.length > 0 ? (
        <div className="w-auto h-auto space-x-3 space-y-5 flex flex-wrap justify-start items-center">
          {videos.map((video, i) => (
            <div
              key={i}
              className="w-80 aspect-video cursor-pointer"
              onClick={() => handleOnClick(video)}
            >
              {video.status === "pending" ? (
                <div className="w-full h-full bg-[#3d3d3d] flex justify-center items-center">
                  <p>In Queue</p>
                </div>
              ) : (
                <img src={video.videoUrls.poster} style={{ width: "100%" ,height:"100%"}} />
              )}
              <p className="w-full text-sm text-wrap my-4">{video.title}</p>
            </div>
          ))}
        </div>
      ) : (
        <div className="w-full h-3/4 flex justify-center items-center">
          <p className="text-sm text-wrap my-10">No videos found</p>
        </div>
      )}
    </div>
  );
};

export default VideoList;
