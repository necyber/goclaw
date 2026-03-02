import React from "react";
import ReactDOM from "react-dom/client";

import App from "./App";
import "./index.css";
import { initializeUIBasePath } from "./lib/uiBasePath";

initializeUIBasePath(import.meta.url);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
