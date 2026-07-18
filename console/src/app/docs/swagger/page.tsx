"use client";
import { useEffect } from "react";
export default function SwaggerPage() {
  useEffect(() => { window.location.href = "/docs"; }, []);
  return null;
}
