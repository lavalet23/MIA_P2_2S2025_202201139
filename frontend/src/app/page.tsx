"use client";
import { useState } from "react";
import InputTerminal from "@/components/InputTerminal";
import OutputTerminal from "@/components/OutputTerminal";
import FileUpload from "@/components/FileUpload";
import { executeCommands } from "@/services/api";

export default function Home() {
  // === Estados generales ===
  const [input, setInput] = useState("");
  const [output, setOutput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<"ejecucion" | "login">("ejecucion");

  // === Estados del login simulado ===
  const [loginUser, setLoginUser] = useState("");
  const [loginPass, setLoginPass] = useState("");
  const [loginId, setLoginId] = useState("");
  const [loginMessage, setLoginMessage] = useState<string | null>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  // === Funciones de la consola ===
  const handleClear = () => {
    setInput("");
    setOutput("");
  };

  const handleExecute = async () => {
    if (!input.trim()) {
      setOutput("Ingrese los comandos...");
      return;
    }

    setIsLoading(true);
    try {
      const lines = input.split("\n");
      let outputResult = "";

      for (const line of lines) {
        const trimmedLine = line.trim();
        if (trimmedLine && !trimmedLine.startsWith("#")) {
          const result = await executeCommands(trimmedLine);
          if (result) outputResult += result + "\n";
        }
      }

      setOutput(outputResult);
    } catch (error) {
      setOutput(error instanceof Error ? error.message : "Error desconocido");
    } finally {
      setIsLoading(false);
    }
  };

  const handleFileContent = (content: string) => {
    setInput(content);
  };

  // === Simulación del inicio/cierre de sesión ===
  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();

    // Credenciales correctas
    const validUser = "root";
    const validPass = "123";
    const validId = "391A";

    if (loginUser === validUser && loginPass === validPass && loginId === validId) {
      setIsLoggedIn(true);
      setLoginMessage("✅ Inicio de sesión exitoso. Bienvenido root.");
    } else {
      setIsLoggedIn(false);
      setLoginMessage("❌ Credenciales incorrectas. Inténtalo nuevamente.");
    }
  };

  const handleLogout = () => {
    setIsLoggedIn(false);
    setLoginUser("");
    setLoginPass("");
    setLoginId("");
    setLoginMessage("🔒 Sesión cerrada correctamente.");
  };

  // === Render principal ===
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* === ENCABEZADO === */}
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-2xl font-bold text-gray-800 dark:text-white">
            GoDisk
          </h1>

          <div className="flex gap-2">
            <button
              onClick={() => setActiveTab("ejecucion")}
              className={`px-4 py-2 rounded-lg font-medium ${
                activeTab === "ejecucion"
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 text-gray-700 hover:bg-gray-300"
              }`}
            >
              🖥️ Ejecución
            </button>
            <button
              onClick={() => setActiveTab("login")}
              className={`px-4 py-2 rounded-lg font-medium ${
                activeTab === "login"
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 text-gray-700 hover:bg-gray-300"
              }`}
            >
              🔐 Login
            </button>
          </div>
        </div>

        {/* === CONTENIDO DE PESTAÑAS === */}
        {activeTab === "ejecucion" ? (
          <>
            {/* === CONSOLA DE EJECUCIÓN === */}
            <div className="flex justify-end items-center mb-8 gap-4">
              <label
                htmlFor="file-upload"
                className="px-4 py-2 rounded-lg bg-blue-500 text-white hover:bg-blue-600 
                          transition-colors duration-200 cursor-pointer shadow-sm
                          flex items-center gap-2"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
                  />
                </svg>
                Cargar Archivo
              </label>

              <button
                onClick={handleClear}
                className="px-4 py-2 rounded-lg bg-red-500 text-white 
                hover:bg-red-600 transition-colors duration-200 
                shadow-sm flex items-center gap-2"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
                Limpiar
              </button>

              <button
                onClick={handleExecute}
                disabled={isLoading}
                className={`px-4 py-2 rounded-lg bg-green-500 text-white 
                hover:bg-green-600 transition-colors duration-200 
                shadow-sm flex items-center gap-2
                ${isLoading ? "opacity-50 cursor-not-allowed" : ""}`}
              >
                {isLoading ? "Ejecutando..." : "Ejecutar"}
              </button>
            </div>

            <FileUpload onFileContent={handleFileContent} />
            <InputTerminal value={input} onChange={setInput} />
            <OutputTerminal output={output} />
          </>
        ) : (
          /* === LOGIN SIMULADO === */
          <div className="flex flex-col items-center justify-center py-12">
            <h2 className="text-2xl font-semibold mb-6 text-gray-800 dark:text-white">
              {isLoggedIn ? "Sesión Activa" : "Iniciar Sesión"}
            </h2>

            {!isLoggedIn ? (
              <form
                onSubmit={handleLogin}
                className="flex flex-col gap-4 w-full max-w-sm bg-white dark:bg-gray-800 p-6 rounded-lg shadow"
              >
                <input
                  type="text"
                  placeholder="ID Partición"
                  value={loginId}
                  onChange={(e) => setLoginId(e.target.value)}
                  className="p-2 border rounded"
                  required
                />
                <input
                  type="text"
                  placeholder="Usuario"
                  value={loginUser}
                  onChange={(e) => setLoginUser(e.target.value)}
                  className="p-2 border rounded"
                  required
                />
                <input
                  type="password"
                  placeholder="Contraseña"
                  value={loginPass}
                  onChange={(e) => setLoginPass(e.target.value)}
                  className="p-2 border rounded"
                  required
                />
                <button
                  type="submit"
                  className="bg-blue-600 hover:bg-blue-700 text-white py-2 rounded"
                >
                  Iniciar sesión
                </button>

                {loginMessage && (
                  <div
                    className={`text-center font-medium mt-2 ${
                      loginMessage.includes("✅")
                        ? "text-green-600"
                        : "text-red-500"
                    }`}
                  >
                    {loginMessage}
                  </div>
                )}
              </form>
            ) : (
              <div className="flex flex-col items-center gap-4 bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
                <p className="text-green-600 font-semibold">
                  ✅ Sesión iniciada correctamente como <b>root</b>.
                </p>
                <button
                  onClick={handleLogout}
                  className="bg-red-600 hover:bg-red-700 text-white py-2 px-4 rounded"
                >
                  Cerrar sesión
                </button>

                {loginMessage && (
                  <p className="text-gray-600 mt-2">{loginMessage}</p>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
