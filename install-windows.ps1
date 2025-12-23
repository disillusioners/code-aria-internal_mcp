# PowerShell script to install MCP servers on Windows

# Check if all executables exist
$requiredFiles = @("mcp-filesystem.exe", "mcp-codebase.exe", "mcp-git.exe", "mcp-code-edit.exe", "mcp-bash.exe", "mcp-powershell.exe", "mcp-systeminfo.exe", "mcp-savepoints.exe", "mcp-lang-go.exe", "mcp-guidelines.exe", "mcp-documents.exe", "mcp-postgres.exe")
foreach ($file in $requiredFiles) {
    if (-not (Test-Path $file)) {
        Write-Error "Error: MCP servers not built. Run 'make build-mcp-servers' first."
        exit 1
    }
}

Write-Host "Checking for installation directory..."

# Try to find or create installation directory
$binDir = Join-Path $env:USERPROFILE 'bin'
if (Test-Path $binDir) {
    Write-Host "Found existing directory: $binDir"
    $installDir = $binDir
} else {
    try {
        New-Item -Path $binDir -ItemType Directory -ErrorAction Stop | Out-Null
        Write-Host "Created directory: $binDir"
        $installDir = $binDir
    } catch {
        $localDir = Join-Path $env:USERPROFILE '.local\bin'
        if (Test-Path $localDir) {
            Write-Host "Found existing directory: $localDir"
            $installDir = $localDir
        } else {
            try {
                New-Item -Path $localDir -ItemType Directory -ErrorAction Stop | Out-Null
                Write-Host "Created directory: $localDir"
                $installDir = $localDir
            } catch {
                Write-Error "Error: No writable installation directory found."
                Write-Error "Please create one of: $binDir or $localDir."
                exit 1
            }
        }
    }
}

Write-Host "Installing to $installDir"

# Remove existing MCP server executables from destination
Write-Host "Removing existing MCP server executables from $installDir..."
$allExecutables = @("mcp-filesystem.exe", "mcp-codebase.exe", "mcp-git.exe", "mcp-code-edit.exe", "mcp-bash.exe", "mcp-powershell.exe", "mcp-systeminfo.exe", "mcp-savepoints.exe", "mcp-lang-go.exe", "mcp-guidelines.exe", "mcp-documents.exe", "mcp-postgres.exe")
foreach ($exe in $allExecutables) {
    $destPath = Join-Path $installDir $exe
    if (Test-Path $destPath) {
        Remove-Item $destPath -Force -ErrorAction SilentlyContinue
    }
}

# Copy all executables
foreach ($file in $requiredFiles) {
    Copy-Item $file $installDir
}

Write-Host "MCP servers installed successfully to $installDir"

# Check if directory is in PATH
if ($env:PATH -notlike "*$installDir*") {
    Write-Host ""
    Write-Host "Warning: $installDir is not in your PATH."
    Write-Host "Add this to your environment variables:"
    Write-Host "  set PATH=$installDir;%PATH%"
}

# Output the installation directory for the Makefile
Write-Output $installDir