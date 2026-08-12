export interface ErrorEnvelope {
  error: {
    code: string;
    message: string;
    requestId: string;
  };
}

export type ApiResponse<T> = T; // Assuming success response is direct resource
