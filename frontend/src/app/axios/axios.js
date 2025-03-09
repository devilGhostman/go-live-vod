import axios from "axios";

const API = axios.create({
  baseURL: `${process.env.NEXT_PUBLIC_API_BASE_URL}/`,
  // baseURL: `http://0.0.0.0:8000/`,
});

export default API;
