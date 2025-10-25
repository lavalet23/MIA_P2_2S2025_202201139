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
  const [activeTab, setActiveTab] = useState<
    "ejecucion" | "login" | "explorador"
  >("ejecucion");

  // === Estados del login ===
  const [loginUser, setLoginUser] = useState("");
  const [loginPass, setLoginPass] = useState("");
  const [loginId, setLoginId] = useState("");
  const [loginMessage, setLoginMessage] = useState<string | null>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  // === Estado del explorador ===
  const [filesystem, setFilesystem] = useState<any>({ disks: [] });
  const [directories, setDirectories] = useState<any[]>([
    { name: "/", type: "folder", children: [] },
  ]);

  // === Funciones generales ===
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
      parseOutputToFilesystem(outputResult); // 🔄 Actualiza explorador
    } catch (error) {
      setOutput(error instanceof Error ? error.message : "Error desconocido");
    } finally {
      setIsLoading(false);
    }
  };

  const handleFileContent = (content: string) => {
    setInput(content);
  };

  // === Login simulado ===
  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();

    const validCredentials = [
      { user: "root", pass: "123", id: "391a" },
      { user: "user1", pass: "abc", id: "391A" },
    ];

    const isValid = validCredentials.some(
      (cred) =>
        cred.user === loginUser && cred.pass === loginPass && cred.id === loginId
    );

    if (isValid) {
      setIsLoggedIn(true);
      setLoginMessage(`✅ Inicio de sesión exitoso. Bienvenido ${loginUser}.`);
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

  // === Parsear salida y actualizar el explorador ===
  const parseOutputToFilesystem = (output: string) => {
    const lines = output.split("\n");
    const newFilesystem = { ...filesystem };
    const newDirs = JSON.parse(JSON.stringify(directories));

    const findOrCreatePath = (path: string, type: "folder" | "file") => {
      const parts = path.split("/").filter(Boolean);
      let current = newDirs[0];
      for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        let child = current.children.find((c: any) => c.name === part);
        if (!child) {
          child = {
            name: part,
            type: i === parts.length - 1 ? type : "folder",
            children: [],
          };
          current.children.push(child);
        }
        current = child;
      }
    };

    const removePath = (path: string) => {
      const parts = path.split("/").filter(Boolean);
      const target = parts.pop();
      let current = newDirs[0];
      for (const part of parts) {
        const next = current.children.find((c: any) => c.name === part);
        if (!next) return;
        current = next;
      }
      current.children = current.children.filter((c: any) => c.name !== target);
    };

    const renamePath = (path: string, newName: string) => {
      const parts = path.split("/").filter(Boolean);
      const target = parts.pop();
      let current = newDirs[0];
      for (const part of parts) {
        const next = current.children.find((c: any) => c.name === part);
        if (!next) return;
        current = next;
      }
      const item = current.children.find((c: any) => c.name === target);
      if (item) item.name = newName;
    };

    const movePath = (path: string, dest: string) => {
      const parts = path.split("/").filter(Boolean);
      const target = parts.pop();
      let current = newDirs[0];
      for (const part of parts) {
        const next = current.children.find((c: any) => c.name === part);
        if (!next) return;
        current = next;
      }
      const item = current.children.find((c: any) => c.name === target);
      if (!item) return;
      current.children = current.children.filter((c: any) => c.name !== target);
      const destParts = dest.split("/").filter(Boolean);
      let destNode = newDirs[0];
      for (const part of destParts) {
        let next = destNode.children.find((c: any) => c.name === part);
        if (!next) {
          next = { name: part, type: "folder", children: [] };
          destNode.children.push(next);
        }
        destNode = next;
      }
      destNode.children.push(item);
    };

    lines.forEach((line) => {
      line = line.trim();

      // === CREAR DISCO ===
      if (line.startsWith("MKDISK: Disco creado exitosamente")) {
        const pathLine = lines[lines.indexOf(line) + 1];
        const sizeLine = lines[lines.indexOf(line) + 2];
        const matchPath = pathLine?.match(/-> Path: (.*)/);
        const matchSize = sizeLine?.match(/-> Tamaño: (.*)/);
        if (matchPath && matchSize) {
          const name = matchPath[1].split("/").pop();
          if (!newFilesystem.disks.some((d: any) => d.path === matchPath[1])) {
            newFilesystem.disks.push({
              name,
              path: matchPath[1],
              size: matchSize[1],
              partitions: [],
            });
          }
        }
      }

      // === ELIMINAR DISCO ===
      if (line.startsWith("RMDISK: Disco eliminado correctamente")) {
        const match = line.match(/-> Path: (.*)/);
        if (match) {
          const path = match[1];
          newFilesystem.disks = newFilesystem.disks.filter(
            (d: any) => d.path !== path
          );
        }
      }

      // === CREAR PARTICIÓN ===
      if (line.startsWith("FDISK: Partición") && line.includes("creada exitosamente")) {
        const nameMatch = line.match(/'(.+)'/);
        const sizeLine = lines[lines.indexOf(line) + 1];
        const matchSize = sizeLine?.match(/-> Tamaño: (.*)/);
        if (nameMatch && matchSize) {
          const partName = nameMatch[1];
          const partSize = matchSize[1];
          const lastDisk = newFilesystem.disks[newFilesystem.disks.length - 1];
          if (lastDisk) {
            const exists = lastDisk.partitions.some(
              (p: any) => p.name === partName
            );
            if (!exists) {
              lastDisk.partitions.push({ name: partName, size: partSize });
            }
          }
        }
      }

      // === CREAR DIRECTORIO ===
      if (line.startsWith("MKDIR: Directorio")) {
        const match = line.match(/\/[\w/]+/);
        if (match) findOrCreatePath(match[0], "folder");
      }

      // === CREAR ARCHIVO ===
      // === CREAR ARCHIVO ===
if (line.startsWith("MKFILE: Archivo creado exitosamente")) {
  const nextLine = lines[lines.indexOf(line) + 1];
  const match = nextLine?.match(/-> Path: (.*)/);
  if (match) {
    const filePath = match[1].trim();
    findOrCreatePath(filePath, "file");
  }
}


      // === REMOVE ===
      if (line.startsWith("REMOVE:")) {
        const match = line.match(/-> (.*)/);
        if (match) removePath(match[1]);
      }

      // === RENAME ===
      if (line.startsWith("RENAME:")) {
        const pathMatch = line.match(/-> Path: (.*)/);
        const nameMatch = line.match(/-> Nuevo nombre: (.*)/);
        if (pathMatch && nameMatch) renamePath(pathMatch[1], nameMatch[1]);
      }

      // === MOVE ===
      if (line.startsWith("MOVE:")) {
        const pathMatch = line.match(/-> Origen: (.*)/);
        const destMatch = line.match(/-> Destino: (.*)/);
        if (pathMatch && destMatch) movePath(pathMatch[1], destMatch[1]);
      }
    });

    setFilesystem(newFilesystem);
    setDirectories(newDirs);
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

            <button
              onClick={() => setActiveTab("explorador")}
              className={`px-4 py-2 rounded-lg font-medium ${
                activeTab === "explorador"
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 text-gray-700 hover:bg-gray-300"
              }`}
            >
              🗂️ Explorador
            </button>
          </div>
        </div>

        {/* === Pestañas === */}
        {activeTab === "ejecucion" && (
          <>
            <div className="flex justify-end items-center mb-8 gap-4">
              <label
                htmlFor="file-upload"
                className="px-4 py-2 rounded-lg bg-blue-500 text-white hover:bg-blue-600 transition-colors duration-200 cursor-pointer shadow-sm flex items-center gap-2"
              >
                📂 Cargar Archivo
              </label>

              <button
                onClick={handleClear}
                className="px-4 py-2 rounded-lg bg-red-500 text-white hover:bg-red-600 transition-colors duration-200 shadow-sm flex items-center gap-2"
              >
                🧹 Limpiar
              </button>

              <button
                onClick={handleExecute}
                disabled={isLoading}
                className={`px-4 py-2 rounded-lg bg-green-500 text-white hover:bg-green-600 transition-colors duration-200 shadow-sm flex items-center gap-2 ${
                  isLoading ? "opacity-50 cursor-not-allowed" : ""
                }`}
              >
                {isLoading ? "⏳ Ejecutando..." : "▶️ Ejecutar"}
              </button>
            </div>

            <FileUpload onFileContent={handleFileContent} />
            <InputTerminal value={input} onChange={setInput} />
            <OutputTerminal output={output} />
          </>
        )}

        {activeTab === "login" && (
          <LoginTab
            isLoggedIn={isLoggedIn}
            loginUser={loginUser}
            loginMessage={loginMessage}
            handleLogin={handleLogin}
            handleLogout={handleLogout}
            setLoginUser={setLoginUser}
            setLoginPass={setLoginPass}
            setLoginId={setLoginId}
            loginId={loginId}
            loginPass={loginPass}
          />
        )}

        {activeTab === "explorador" && (
          <ExplorerTab
            filesystem={filesystem}
            directories={directories}
            parseOutputToFilesystem={parseOutputToFilesystem}
            output={output}
          />
        )}
      </div>
    </div>
  );
}

// === Componente: Explorador ===
const ExplorerTab = ({
  filesystem,
  directories,
  parseOutputToFilesystem,
  output,
}: any) => (
  <div className="p-6 bg-white dark:bg-gray-800 rounded-lg shadow-md">
    <h2 className="text-xl font-semibold mb-4 text-gray-800 dark:text-white">
      🗂️ Explorador de archivos
    </h2>
    <button
      onClick={() => parseOutputToFilesystem(output)}
      className="mb-4 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded"
    >
      🔄 Actualizar
    </button>
    {filesystem.disks.length === 0 ? (
      <p className="text-gray-500">Aún no hay discos creados.</p>
    ) : (
      <div className="space-y-6">
        {filesystem.disks.map((disk: any, i: number) => (
          <div
            key={i}
            className="border rounded-lg p-3 bg-gray-50 dark:bg-gray-700"
          >
            <h3 className="font-bold text-gray-800 dark:text-white">
              💿 {disk.name}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-300">
              Ruta: {disk.path}
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
              Tamaño: {disk.size}
            </p>
            {disk.partitions.length > 0 ? (
              <ul className="pl-6 list-disc text-gray-700 dark:text-gray-200 mb-4">
                {disk.partitions.map((p: any, j: number) => (
                  <li key={j}>🧩 {p.name} — {p.size}</li>
                ))}
              </ul>
            ) : (
              <p className="text-gray-500 mb-4">Sin particiones.</p>
            )}
            {/* === Mostrar árbol solo en Disco1.mia === */}
            {disk.name === "Disco1.mia" && (
              <div className="font-mono text-sm text-gray-800 dark:text-gray-100 ml-4 border-l-2 border-gray-400 pl-3">
                <FileTree node={directories[0]} level={0} />
              </div>
            )}
          </div>
        ))}
      </div>
    )}
  </div>
);

// === Componente: Árbol de archivos bonito ===
const FileTree = ({ node, level }: { node: any; level: number }) => {
  const color =
    node.type === "folder" ? "text-yellow-600" : "text-blue-500 font-semibold";
  const icon = node.type === "folder" ? "📁" : "📄";

  return (
    <div className={`ml-${level * 2}`}>
      <div className={`flex items-center gap-2 ${color}`}>
        <span>{icon}</span>
        <span>{node.name}</span>
      </div>
      {node.children &&
        node.children.map((child: any) => (
          <div key={child.name} className="ml-4">
            <FileTree node={child} level={level + 1} />
          </div>
        ))}
    </div>
  );
};

// === Componente: Login ===
const LoginTab = ({
  isLoggedIn,
  loginUser,
  loginMessage,
  handleLogin,
  handleLogout,
  setLoginUser,
  setLoginPass,
  setLoginId,
  loginId,
  loginPass,
}: any) => (
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
              loginMessage.includes("✅") ? "text-green-600" : "text-red-500"
            }`}
          >
            {loginMessage}
          </div>
        )}
      </form>
    ) : (
      <div className="flex flex-col items-center gap-4 bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
        <p className="text-green-600 font-semibold">
          ✅ Sesión iniciada correctamente como <b>{loginUser}</b>.
        </p>
        <button
          onClick={handleLogout}
          className="bg-red-600 hover:bg-red-700 text-white py-2 px-4 rounded"
        >
          Cerrar sesión
        </button>
      </div>
    )}
  </div>
);
