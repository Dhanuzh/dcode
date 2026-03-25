use crate::ManagedPackage;
use std::path::PathBuf;

/// Immutable configuration for a [`crate::PackageManager`] instance.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PackageManagerConfig<P> {
    pub(crate) dcode_home: PathBuf,
    pub(crate) package: P,
    cache_root: Option<PathBuf>,
}

impl<P> PackageManagerConfig<P> {
    /// Creates a config rooted at the provided Dcode home directory.
    pub fn new(dcode_home: PathBuf, package: P) -> Self {
        Self {
            dcode_home,
            package,
            cache_root: None,
        }
    }

    /// Overrides the package cache root instead of deriving it from `dcode_home`.
    pub fn with_cache_root(mut self, cache_root: PathBuf) -> Self {
        self.cache_root = Some(cache_root);
        self
    }
}

impl<P: ManagedPackage> PackageManagerConfig<P> {
    /// Returns the effective cache root for the package.
    pub fn cache_root(&self) -> PathBuf {
        self.cache_root.clone().unwrap_or_else(|| {
            self.dcode_home.join(
                self.package
                    .default_cache_root_relative()
                    .replace('/', std::path::MAIN_SEPARATOR_STR),
            )
        })
    }
}
