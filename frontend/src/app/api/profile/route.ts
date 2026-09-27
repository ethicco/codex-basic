import { NextRequest } from "next/server";

import { handleProfile } from "@/lib/auth-proxy";

export async function GET(request: NextRequest) {
  return handleProfile(request);
}

export async function POST(request: NextRequest) {
  return handleProfile(request);
}

export async function PATCH(request: NextRequest) {
  return handleProfile(request);
}
