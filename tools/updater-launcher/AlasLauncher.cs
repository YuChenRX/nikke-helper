using System;
using System.Diagnostics;
using System.IO;

internal static class AlasLauncher
{
    private static int Main()
    {
        string baseDir = AppDomain.CurrentDomain.BaseDirectory;
        string scriptPath = Path.Combine(baseDir, "update-and-start.ps1");

        if (!File.Exists(scriptPath))
        {
            scriptPath = Path.Combine(baseDir, "release", "update-and-start.ps1");
        }

        if (!File.Exists(scriptPath))
        {
            return StartApp(baseDir);
        }

        string shell = FindPowerShell();
        string arguments = "-NoProfile -ExecutionPolicy Bypass -File \"" + scriptPath + "\"";

        try
        {
            using (Process process = Process.Start(new ProcessStartInfo
            {
                FileName = shell,
                Arguments = arguments,
                WorkingDirectory = Path.GetDirectoryName(scriptPath) ?? baseDir,
                UseShellExecute = false,
            }))
            {
                process.WaitForExit();
                return process.ExitCode;
            }
        }
        catch
        {
            return StartApp(baseDir);
        }
    }

    private static string FindPowerShell()
    {
        string programFiles = Environment.GetFolderPath(Environment.SpecialFolder.ProgramFiles);
        string pwsh = Path.Combine(programFiles, "PowerShell", "7", "pwsh.exe");
        return File.Exists(pwsh) ? pwsh : "powershell.exe";
    }

    private static int StartApp(string baseDir)
    {
        string[] candidates =
        {
            Path.Combine(baseDir, "alas-app.exe"),
            Path.Combine(baseDir, "release", "alas-app.exe"),
            Path.Combine(baseDir, "release", "alas.exe"),
        };

        foreach (string candidate in candidates)
        {
            if (!File.Exists(candidate))
            {
                continue;
            }

            try
            {
                Process.Start(new ProcessStartInfo
                {
                    FileName = candidate,
                    WorkingDirectory = Path.GetDirectoryName(candidate) ?? baseDir,
                    UseShellExecute = true,
                });
                return 0;
            }
            catch
            {
                return 1;
            }
        }

        return 1;
    }
}
