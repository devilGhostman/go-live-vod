"use client";

import React, { useCallback, useState } from "react";

import axios from "../axios/axios";
import Navbar from "../components/navbar";
import { useRouter } from "next/navigation";

function Upload() {
  const router = useRouter();
  const [error, seterror] = useState("");
  const [onSuccess, setonSuccess] = useState(false);

  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [selectedResolutions, setSelectedResolutions] = useState([]);

  const handleResolutionToggle = (resolution) => {
    setSelectedResolutions((prev) =>
      prev.includes(resolution)
        ? prev.filter((res) => res !== resolution)
        : [...prev, resolution]
    );
  };
  const handleSubmit = (e) => {
    e.preventDefault();
    console.log({
      name,
      url,
      resolutions: selectedResolutions,
    });
    handleUpload(name, url, selectedResolutions);
  };

  const generateRandomId = () => {
    return `id-${Math.random().toString(36).slice(2, 11)}-${Date.now()}`;
  };

  const handleUpload = async (name, url, resolutions) => {
    try {
      const response = await axios.post("/vod/", {
        id: generateRandomId(),
        namespace: "dev",
        streamName: name,
        sourceUrl: url,
        config: {
          codecs: {
            video: "h264",
            audio: "aac",
          },
          resolutions: resolutions,
        },
      });

      if (response.status === 200) {
        router.push("/");
      }
      console.log(response.data);
    } catch (error) {
      console.error(error);
      seterror(error.message);
    }
  };

  return (
    <>
      <Navbar />
      <div className="w-full h-screen flex justify-center items-center flex-col">
        <form onSubmit={handleSubmit}>
          {/* Name Input */}
          <div className="mb-4">
            <label
              htmlFor="name"
              className="block text-sm font-medium text-gray-700"
            >
              Name
            </label>
            <input
              type="text"
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="mt-1 p-2 w-full border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder="Enter name"
              required
            />
          </div>

          {/* URL Input */}
          <div className="mb-4">
            <label
              htmlFor="url"
              className="block text-sm font-medium text-gray-700"
            >
              URL
            </label>
            <input
              type="url"
              id="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              className="mt-1 p-2 w-full border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder="Enter URL"
              required
            />
          </div>

          {/* Resolutions as Buttons */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700">
              Resolutions
            </label>
            <div className="flex gap-2 flex-wrap">
              {["144p", "240p", "360p", "480p", "720p", "1080p"].map(
                (resolution) => (
                  <button
                    key={resolution}
                    type="button"
                    onClick={() => handleResolutionToggle(resolution)}
                    className={`px-4 py-2 rounded-full text-sm font-medium transition-all focus:outline-none ${
                      selectedResolutions.includes(resolution)
                        ? "bg-red-700  text-white"
                        : "bg-gray-200 text-gray-700"
                    } hover:bg-red-500 hover:text-white`}
                  >
                    {resolution}
                  </button>
                )
              )}
            </div>
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            className="mt-5 text-[18px] font-bold hover:cursor-pointer hover:text-red-500 w-1/6"
          >
            {onSuccess ? "VIDEO UPLOADED" : "UPLOAD"}
          </button>
        </form>
        {error && <p>{error}</p>}
      </div>
    </>
  );
}

export default Upload;
